package providers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/bpfs/dep2p/internal"

	lru "github.com/hashicorp/golang-lru/simplelru"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/autobatch"
	dsq "github.com/ipfs/go-datastore/query"

	"github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	peerstoreImpl "github.com/libp2p/go-libp2p/p2p/host/peerstore"
	"github.com/multiformats/go-base32"
)

const (
	ProvidersKeyPrefix = "/providers/"

	ProviderAddrTTL = 24 * time.Hour
)

var ProvideValidity = time.Hour * 48
var defaultCleanupInterval = time.Hour
var lruCacheSize = 256
var batchBufferSize = 256

// ProviderStore 表示将节点及其地址与密钥相关联的存储。
type ProviderStore interface {
	AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error
	GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error)
	io.Closer
}

// ProviderManager 在数据存储中添加和拉出提供者，并在两者之间缓存它们
type ProviderManager struct {
	self peer.ID

	cache  lru.LRUCache
	pstore peerstore.Peerstore
	dstore *autobatch.Datastore

	newprovs chan *addProv
	getprovs chan *getProv

	cleanupInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var _ ProviderStore = (*ProviderManager)(nil)

// Option 是设置提供者管理器选项的函数。
type Option func(*ProviderManager) error

func (pm *ProviderManager) applyOptions(opts ...Option) error {
	for i, opt := range opts {
		if err := opt(pm); err != nil {
			return fmt.Errorf("provider manager option %d failed: %s", i, err)
		}
	}
	return nil
}

// CleanupInterval 设置 GC 运行之间的时间。
// 默认为 1 小时。
func CleanupInterval(d time.Duration) Option {
	return func(pm *ProviderManager) error {
		pm.cleanupInterval = d
		return nil
	}
}

// Cache 设置 LRU 缓存实现。
// 默认为简单的 LRU 缓存。
func Cache(c lru.LRUCache) Option {
	return func(pm *ProviderManager) error {
		pm.cache = c
		return nil
	}
}

type addProv struct {
	ctx context.Context
	key []byte
	val peer.ID
}

type getProv struct {
	ctx  context.Context
	key  []byte
	resp chan []peer.ID
}

func NewProviderManager(local peer.ID, ps peerstore.Peerstore, dstore ds.Batching, opts ...Option) (*ProviderManager, error) {
	pm := new(ProviderManager)
	pm.self = local
	pm.getprovs = make(chan *getProv)
	pm.newprovs = make(chan *addProv)
	pm.pstore = ps
	pm.dstore = autobatch.NewAutoBatching(dstore, batchBufferSize)
	cache, err := lru.NewLRU(lruCacheSize, nil)
	if err != nil {
		return nil, err
	}
	pm.cache = cache
	pm.cleanupInterval = defaultCleanupInterval
	if err := pm.applyOptions(opts...); err != nil {
		return nil, err
	}
	pm.ctx, pm.cancel = context.WithCancel(context.Background())
	pm.run()
	return pm, nil
}

func (pm *ProviderManager) run() {
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()

		var gcQuery dsq.Results
		gcTimer := time.NewTimer(pm.cleanupInterval)

		defer func() {
			gcTimer.Stop()
			if gcQuery != nil {
				_ = gcQuery.Close()
			}
			if err := pm.dstore.Flush(context.Background()); err != nil {
				logrus.Errorln("failed to flush datastore: ", err)
			}
		}()

		var gcQueryRes <-chan dsq.Result
		var gcSkip map[string]struct{}
		var gcTime time.Time
		for {
			select {
			case np := <-pm.newprovs:
				err := pm.addProv(np.ctx, np.key, np.val)
				if err != nil {
					logrus.Errorln("error adding new providers: ", err)
					continue
				}
				if gcSkip != nil {
					gcSkip[mkProvKeyFor(np.key, np.val)] = struct{}{}
				}
			case gp := <-pm.getprovs:
				provs, err := pm.getProvidersForKey(gp.ctx, gp.key)
				if err != nil && err != ds.ErrNotFound {
					logrus.Errorln("error reading providers: ", err)
				}

				gp.resp <- provs[0:len(provs):len(provs)]
			case res, ok := <-gcQueryRes:
				if !ok {
					if err := gcQuery.Close(); err != nil {
						logrus.Errorln("failed to close provider GC query: ", err)
					}
					gcTimer.Reset(pm.cleanupInterval)

					gcQueryRes = nil
					gcSkip = nil
					gcQuery = nil
					continue
				}
				if res.Error != nil {
					logrus.Errorln("got error from GC query: ", res.Error)
					continue
				}
				if _, ok := gcSkip[res.Key]; ok {
					continue
				}

				t, err := readTimeValue(res.Value)
				switch {
				case err != nil:
					logrus.Errorln("parsing providers record from disk: ", err)
					fallthrough
				case gcTime.Sub(t) > ProvideValidity:
					err = pm.dstore.Delete(pm.ctx, ds.RawKey(res.Key))
					if err != nil && err != ds.ErrNotFound {
						logrus.Errorln("failed to remove provider record from disk: ", err)
					}
				}

			case gcTime = <-gcTimer.C:
				pm.cache.Purge()

				q, err := pm.dstore.Query(pm.ctx, dsq.Query{
					Prefix: ProvidersKeyPrefix,
				})
				if err != nil {
					logrus.Errorln("provider record GC query failed: ", err)
					continue
				}
				gcQuery = q
				gcQueryRes = q.Next()
				gcSkip = make(map[string]struct{})
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

func (pm *ProviderManager) Close() error {
	pm.cancel()
	pm.wg.Wait()
	return nil
}

// AddProvider 添加提供商
func (pm *ProviderManager) AddProvider(ctx context.Context, k []byte, provInfo peer.AddrInfo) error {
	ctx, span := internal.StartSpan(ctx, "ProviderManager.AddProvider")
	defer span.End()

	if provInfo.ID != pm.self {
		pm.pstore.AddAddrs(provInfo.ID, provInfo.Addrs, ProviderAddrTTL)
	}
	prov := &addProv{
		ctx: ctx,
		key: k,
		val: provInfo.ID,
	}
	select {
	case pm.newprovs <- prov:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// addProv 如果需要更新缓存
func (pm *ProviderManager) addProv(ctx context.Context, k []byte, p peer.ID) error {
	now := time.Now()
	if provs, ok := pm.cache.Get(string(k)); ok {
		provs.(*providerSet).setVal(p, now)
	}

	return writeProviderEntry(ctx, pm.dstore, k, p, now)
}

// writeProviderEntry 将提供程序写入数据存储区
func writeProviderEntry(ctx context.Context, dstore ds.Datastore, k []byte, p peer.ID, t time.Time) error {
	dsk := mkProvKeyFor(k, p)

	buf := make([]byte, 16)
	n := binary.PutVarint(buf, t.UnixNano())

	return dstore.Put(ctx, ds.NewKey(dsk), buf[:n])
}

func mkProvKeyFor(k []byte, p peer.ID) string {
	return mkProvKey(k) + "/" + base32.RawStdEncoding.EncodeToString([]byte(p))
}

func mkProvKey(k []byte) string {
	return ProvidersKeyPrefix + base32.RawStdEncoding.EncodeToString(k)
}

// GetProviders 返回给定键的提供者集。
// 该方法不复制集合。 不要修改它。
func (pm *ProviderManager) GetProviders(ctx context.Context, k []byte) ([]peer.AddrInfo, error) {
	ctx, span := internal.StartSpan(ctx, "ProviderManager.GetProviders")
	defer span.End()

	gp := &getProv{
		ctx:  ctx,
		key:  k,
		resp: make(chan []peer.ID, 1),
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case pm.getprovs <- gp:
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case peers := <-gp.resp:
		return peerstoreImpl.PeerInfos(pm.pstore, peers), nil
	}
}

func (pm *ProviderManager) getProvidersForKey(ctx context.Context, k []byte) ([]peer.ID, error) {
	pset, err := pm.getProviderSetForKey(ctx, k)
	if err != nil {
		return nil, err
	}
	return pset.providers, nil
}

// 如果 ProviderSet 已存在于缓存中，则返回 ProviderSet，否则从 datastore 加载它
func (pm *ProviderManager) getProviderSetForKey(ctx context.Context, k []byte) (*providerSet, error) {
	cached, ok := pm.cache.Get(string(k))
	if ok {
		return cached.(*providerSet), nil
	}

	pset, err := loadProviderSet(ctx, pm.dstore, k)
	if err != nil {
		return nil, err
	}

	if len(pset.providers) > 0 {
		pm.cache.Add(string(k), pset)
	}

	return pset, nil
}

// 从数据存储中加载 ProviderSet
func loadProviderSet(ctx context.Context, dstore ds.Datastore, k []byte) (*providerSet, error) {
	res, err := dstore.Query(ctx, dsq.Query{Prefix: mkProvKey(k)})
	if err != nil {
		return nil, err
	}
	defer res.Close()

	now := time.Now()
	out := newProviderSet()
	for {
		e, ok := res.NextSync()
		if !ok {
			break
		}
		if e.Error != nil {
			logrus.Errorln("got an error: ", e.Error)
			continue
		}

		t, err := readTimeValue(e.Value)
		switch {
		case err != nil:
			logrus.Errorln("parsing providers record from disk: ", err)
			fallthrough
		case now.Sub(t) > ProvideValidity:
			err = dstore.Delete(ctx, ds.RawKey(e.Key))
			if err != nil && err != ds.ErrNotFound {
				logrus.Errorln("failed to remove provider record from disk: ", err)
			}
			continue
		}

		lix := strings.LastIndex(e.Key, "/")

		decstr, err := base32.RawStdEncoding.DecodeString(e.Key[lix+1:])
		if err != nil {
			logrus.Errorln("base32 decoding error: ", err)
			err = dstore.Delete(ctx, ds.RawKey(e.Key))
			if err != nil && err != ds.ErrNotFound {
				logrus.Errorln("failed to remove provider record from disk: ", err)
			}
			continue
		}

		pid := peer.ID(decstr)

		out.setVal(pid, t)
	}

	return out, nil
}

func readTimeValue(data []byte) (time.Time, error) {
	nsec, n := binary.Varint(data)
	if n <= 0 {
		return time.Time{}, fmt.Errorf("failed to parse time")
	}

	return time.Unix(0, nsec), nil
}
