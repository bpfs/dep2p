// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dht.proto

package dht_pb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	pb "github.com/libp2p/go-libp2p-record/pb"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type Message_MessageType int32

const (
	Message_PUT_VALUE     Message_MessageType = 0
	Message_GET_VALUE     Message_MessageType = 1
	Message_ADD_PROVIDER  Message_MessageType = 2
	Message_GET_PROVIDERS Message_MessageType = 3
	Message_FIND_NODE     Message_MessageType = 4
	Message_PING          Message_MessageType = 5
)

var Message_MessageType_name = map[int32]string{
	0: "PUT_VALUE",
	1: "GET_VALUE",
	2: "ADD_PROVIDER",
	3: "GET_PROVIDERS",
	4: "FIND_NODE",
	5: "PING",
}

var Message_MessageType_value = map[string]int32{
	"PUT_VALUE":     0,
	"GET_VALUE":     1,
	"ADD_PROVIDER":  2,
	"GET_PROVIDERS": 3,
	"FIND_NODE":     4,
	"PING":          5,
}

func (x Message_MessageType) String() string {
	return proto.EnumName(Message_MessageType_name, int32(x))
}

func (Message_MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_616a434b24c97ff4, []int{0, 0}
}

type Message_ConnectionType int32

const (
	// 发送方没有与对等方的连接，并且没有额外信息（默认）
	Message_NOT_CONNECTED Message_ConnectionType = 0
	// 发送者与对等方有实时连接
	Message_CONNECTED Message_ConnectionType = 1
	// 发件人最近连接到对等方
	Message_CAN_CONNECT Message_ConnectionType = 2
	// 发件人最近多次尝试连接对等方但连接失败
	// ("try" here is loose, but this should signal "made strong effort, failed")
	Message_CANNOT_CONNECT Message_ConnectionType = 3
)

var Message_ConnectionType_name = map[int32]string{
	0: "NOT_CONNECTED",
	1: "CONNECTED",
	2: "CAN_CONNECT",
	3: "CANNOT_CONNECT",
}

var Message_ConnectionType_value = map[string]int32{
	"NOT_CONNECTED":  0,
	"CONNECTED":      1,
	"CAN_CONNECT":    2,
	"CANNOT_CONNECT": 3,
}

func (x Message_ConnectionType) String() string {
	return proto.EnumName(Message_ConnectionType_name, int32(x))
}

func (Message_ConnectionType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_616a434b24c97ff4, []int{0, 1}
}

type Message struct {
	ClosestPeers *Message_ClosestPeers `protobuf:"bytes,11,opt,name=closestPeers,proto3" json:"closestPeers,omitempty"`
	// 定义它是什么类型的消息。
	Type Message_MessageType `protobuf:"varint,1,opt,name=type,proto3,enum=dht.pb.Message_MessageType" json:"type,omitempty"`
	// 定义此查询/响应属于哪个珊瑚簇级别。
	// 如果我们将来想实现珊瑚的簇环。
	ClusterLevelRaw int32 `protobuf:"varint,10,opt,name=clusterLevelRaw,proto3" json:"clusterLevelRaw,omitempty"`
	// 用于指定与该消息关联的密钥。
	// PUT_VALUE, GET_VALUE, ADD_PROVIDER, GET_PROVIDERS
	Key []byte `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	// 用于传输附加值
	Value []byte `protobuf:"bytes,12,opt,name=value,proto3" json:"value,omitempty"`
	// 用于返回一个值
	// PUT_VALUE, GET_VALUE
	Record *pb.Record `protobuf:"bytes,3,opt,name=record,proto3" json:"record,omitempty"`
	// 用于返回更接近查询中的键的对等点
	// GET_VALUE, GET_PROVIDERS, FIND_NODE
	CloserPeers []Message_Peer `protobuf:"bytes,8,rep,name=closerPeers,proto3" json:"closerPeers"`
	// 用于返回Provider
	// GET_VALUE, ADD_PROVIDER, GET_PROVIDERS
	ProviderPeers        []Message_Peer `protobuf:"bytes,9,rep,name=providerPeers,proto3" json:"providerPeers"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_616a434b24c97ff4, []int{0}
}
func (m *Message) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return m.Size()
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetClosestPeers() *Message_ClosestPeers {
	if m != nil {
		return m.ClosestPeers
	}
	return nil
}

func (m *Message) GetType() Message_MessageType {
	if m != nil {
		return m.Type
	}
	return Message_PUT_VALUE
}

func (m *Message) GetClusterLevelRaw() int32 {
	if m != nil {
		return m.ClusterLevelRaw
	}
	return 0
}

func (m *Message) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *Message) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Message) GetRecord() *pb.Record {
	if m != nil {
		return m.Record
	}
	return nil
}

func (m *Message) GetCloserPeers() []Message_Peer {
	if m != nil {
		return m.CloserPeers
	}
	return nil
}

func (m *Message) GetProviderPeers() []Message_Peer {
	if m != nil {
		return m.ProviderPeers
	}
	return nil
}

type Message_Peer struct {
	// 给定对等点的 ID。
	Id byteString `protobuf:"bytes,1,opt,name=id,proto3,customtype=byteString" json:"id"`
	// 给定对等点的多地址
	Addrs [][]byte `protobuf:"bytes,2,rep,name=addrs,proto3" json:"addrs,omitempty"`
	// 用于向对等方表明发送方的连接能力
	Connection           Message_ConnectionType `protobuf:"varint,3,opt,name=connection,proto3,enum=dht.pb.Message_ConnectionType" json:"connection,omitempty"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *Message_Peer) Reset()         { *m = Message_Peer{} }
func (m *Message_Peer) String() string { return proto.CompactTextString(m) }
func (*Message_Peer) ProtoMessage()    {}
func (*Message_Peer) Descriptor() ([]byte, []int) {
	return fileDescriptor_616a434b24c97ff4, []int{0, 0}
}
func (m *Message_Peer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message_Peer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message_Peer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message_Peer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message_Peer.Merge(m, src)
}
func (m *Message_Peer) XXX_Size() int {
	return m.Size()
}
func (m *Message_Peer) XXX_DiscardUnknown() {
	xxx_messageInfo_Message_Peer.DiscardUnknown(m)
}

var xxx_messageInfo_Message_Peer proto.InternalMessageInfo

func (m *Message_Peer) GetAddrs() [][]byte {
	if m != nil {
		return m.Addrs
	}
	return nil
}

func (m *Message_Peer) GetConnection() Message_ConnectionType {
	if m != nil {
		return m.Connection
	}
	return Message_NOT_CONNECTED
}

// 新增的 ClosestPeers 数据结构，用于节点发现时握手
type Message_ClosestPeers struct {
	Self                 []byte   `protobuf:"bytes,1,opt,name=self,proto3" json:"self,omitempty"`
	Mode                 int32    `protobuf:"varint,2,opt,name=mode,proto3" json:"mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Message_ClosestPeers) Reset()         { *m = Message_ClosestPeers{} }
func (m *Message_ClosestPeers) String() string { return proto.CompactTextString(m) }
func (*Message_ClosestPeers) ProtoMessage()    {}
func (*Message_ClosestPeers) Descriptor() ([]byte, []int) {
	return fileDescriptor_616a434b24c97ff4, []int{0, 1}
}
func (m *Message_ClosestPeers) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message_ClosestPeers) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message_ClosestPeers.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message_ClosestPeers) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message_ClosestPeers.Merge(m, src)
}
func (m *Message_ClosestPeers) XXX_Size() int {
	return m.Size()
}
func (m *Message_ClosestPeers) XXX_DiscardUnknown() {
	xxx_messageInfo_Message_ClosestPeers.DiscardUnknown(m)
}

var xxx_messageInfo_Message_ClosestPeers proto.InternalMessageInfo

func (m *Message_ClosestPeers) GetSelf() []byte {
	if m != nil {
		return m.Self
	}
	return nil
}

func (m *Message_ClosestPeers) GetMode() int32 {
	if m != nil {
		return m.Mode
	}
	return 0
}

func init() {
	proto.RegisterEnum("dht.pb.Message_MessageType", Message_MessageType_name, Message_MessageType_value)
	proto.RegisterEnum("dht.pb.Message_ConnectionType", Message_ConnectionType_name, Message_ConnectionType_value)
	proto.RegisterType((*Message)(nil), "dht.pb.Message")
	proto.RegisterType((*Message_Peer)(nil), "dht.pb.Message.Peer")
	proto.RegisterType((*Message_ClosestPeers)(nil), "dht.pb.Message.ClosestPeers")
}

func init() { proto.RegisterFile("dht.proto", fileDescriptor_616a434b24c97ff4) }

var fileDescriptor_616a434b24c97ff4 = []byte{
	// 528 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x93, 0x3d, 0x6f, 0x9b, 0x5e,
	0x14, 0xc6, 0xc3, 0x8b, 0xf3, 0x4f, 0x0e, 0xd8, 0x21, 0x57, 0x19, 0x90, 0xff, 0x95, 0x83, 0x3c,
	0xd1, 0xc1, 0x20, 0x51, 0xa9, 0x53, 0x55, 0xc5, 0x36, 0x34, 0xb2, 0x94, 0x62, 0xeb, 0xc6, 0x49,
	0x47, 0xcb, 0xc0, 0x0d, 0x41, 0x25, 0xbe, 0x08, 0xb0, 0x2b, 0x6f, 0xfd, 0x78, 0x19, 0x3b, 0x77,
	0x88, 0x2a, 0x7f, 0x8a, 0x8e, 0xd5, 0xbd, 0xd8, 0x2d, 0x76, 0x87, 0x4e, 0x3c, 0xcf, 0xe1, 0xf9,
	0x89, 0xf3, 0x22, 0xe0, 0x34, 0x7a, 0x2c, 0xad, 0x2c, 0xa7, 0x25, 0x45, 0xc7, 0x5c, 0x06, 0x6d,
	0x27, 0x4e, 0xca, 0xc7, 0x65, 0x60, 0x85, 0xf4, 0xc9, 0x4e, 0x93, 0x20, 0x73, 0x32, 0x3b, 0xa6,
	0xbd, 0x4a, 0xf5, 0x72, 0x12, 0xd2, 0x3c, 0xb2, 0xb3, 0xc0, 0xae, 0x54, 0xc5, 0xb6, 0x7b, 0x35,
	0x26, 0xa6, 0x31, 0xb5, 0x79, 0x39, 0x58, 0x3e, 0x70, 0xc7, 0x0d, 0x57, 0x55, 0xbc, 0xfb, 0xb3,
	0x01, 0xff, 0x7d, 0x24, 0x45, 0x31, 0x8f, 0x09, 0xba, 0x02, 0x35, 0x4c, 0x69, 0x41, 0x8a, 0x72,
	0x42, 0x48, 0x5e, 0xe8, 0x8a, 0x21, 0x98, 0x8a, 0xf3, 0xca, 0xaa, 0xba, 0xb1, 0xb6, 0x31, 0x6b,
	0x58, 0xcb, 0xe0, 0x3d, 0x02, 0xd9, 0x20, 0x97, 0xeb, 0x8c, 0xe8, 0x82, 0x21, 0x98, 0x2d, 0xe7,
	0xff, 0x43, 0x72, 0xfb, 0x9c, 0xae, 0x33, 0x82, 0x79, 0x10, 0x99, 0x70, 0x16, 0xa6, 0xcb, 0xa2,
	0x24, 0xf9, 0x0d, 0x59, 0x91, 0x14, 0xcf, 0xbf, 0xe8, 0x60, 0x08, 0x66, 0x03, 0x1f, 0x96, 0x91,
	0x06, 0xd2, 0x67, 0xb2, 0xd6, 0x45, 0x43, 0x30, 0x55, 0xcc, 0x24, 0xba, 0x80, 0xc6, 0x6a, 0x9e,
	0x2e, 0x89, 0xae, 0xf2, 0x5a, 0x65, 0xd0, 0x6b, 0x38, 0xae, 0xf6, 0xa1, 0x4b, 0xbc, 0xfd, 0x73,
	0x6b, 0xb7, 0x9e, 0xc0, 0xc2, 0x5c, 0xe1, 0x6d, 0x00, 0xbd, 0x03, 0x85, 0x77, 0x9f, 0x57, 0xe3,
	0x9e, 0x18, 0x92, 0xa9, 0x38, 0x17, 0x87, 0x4d, 0xb3, 0x97, 0x03, 0xf9, 0xf9, 0xe5, 0xf2, 0x08,
	0xd7, 0xe3, 0xe8, 0x0a, 0x9a, 0x59, 0x4e, 0x57, 0x49, 0xb4, 0xe3, 0x4f, 0xff, 0xc9, 0xef, 0x03,
	0xed, 0xaf, 0x02, 0xc8, 0x4c, 0xa1, 0x2e, 0x88, 0x49, 0xc4, 0x97, 0xa6, 0x0e, 0x10, 0x4b, 0x7e,
	0x7f, 0xb9, 0x84, 0x60, 0x5d, 0x92, 0xdb, 0x32, 0x4f, 0x16, 0x31, 0x16, 0x93, 0x88, 0x4d, 0x3b,
	0x8f, 0xa2, 0xbc, 0xd0, 0x45, 0x43, 0x62, 0xd3, 0x72, 0x83, 0xde, 0x03, 0x84, 0x74, 0xb1, 0x20,
	0x61, 0x99, 0xd0, 0x05, 0x9f, 0xb8, 0xe5, 0x74, 0xfe, 0x3a, 0xd8, 0xef, 0x04, 0xdf, 0x7c, 0x8d,
	0x68, 0xbf, 0x05, 0xb5, 0x7e, 0x4e, 0x84, 0x40, 0x2e, 0x48, 0xfa, 0x50, 0xf5, 0x82, 0xb9, 0x66,
	0xb5, 0x27, 0x1a, 0x11, 0xbe, 0xfa, 0x06, 0xe6, 0xba, 0x9b, 0x80, 0x52, 0x3b, 0x26, 0x6a, 0xc2,
	0xe9, 0xe4, 0x6e, 0x3a, 0xbb, 0xef, 0xdf, 0xdc, 0x79, 0xda, 0x11, 0xb3, 0xd7, 0xde, 0xce, 0x0a,
	0x48, 0x03, 0xb5, 0xef, 0xba, 0xb3, 0x09, 0x1e, 0xdf, 0x8f, 0x5c, 0x0f, 0x6b, 0x22, 0x3a, 0x87,
	0x26, 0x0b, 0xec, 0x2a, 0xb7, 0x9a, 0xc4, 0x98, 0x0f, 0x23, 0xdf, 0x9d, 0xf9, 0x63, 0xd7, 0xd3,
	0x64, 0x74, 0x02, 0xf2, 0x64, 0xe4, 0x5f, 0x6b, 0x8d, 0xee, 0x27, 0x68, 0xed, 0x0f, 0xc0, 0x68,
	0x7f, 0x3c, 0x9d, 0x0d, 0xc7, 0xbe, 0xef, 0x0d, 0xa7, 0x9e, 0x5b, 0x7d, 0xf1, 0x8f, 0x15, 0xd0,
	0x19, 0x28, 0xc3, 0xbe, 0xbf, 0x4b, 0x68, 0x22, 0x42, 0xd0, 0x1a, 0xf6, 0xfd, 0x1a, 0xa5, 0x49,
	0x03, 0xf5, 0x79, 0xd3, 0x11, 0xbe, 0x6d, 0x3a, 0xc2, 0x8f, 0x4d, 0x47, 0x08, 0x8e, 0xf9, 0xff,
	0xf0, 0xe6, 0x57, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa2, 0x48, 0x6d, 0xe2, 0x87, 0x03, 0x00, 0x00,
}

func (m *Message) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintDht(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x62
	}
	if m.ClosestPeers != nil {
		{
			size, err := m.ClosestPeers.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDht(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x5a
	}
	if m.ClusterLevelRaw != 0 {
		i = encodeVarintDht(dAtA, i, uint64(m.ClusterLevelRaw))
		i--
		dAtA[i] = 0x50
	}
	if len(m.ProviderPeers) > 0 {
		for iNdEx := len(m.ProviderPeers) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.ProviderPeers[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintDht(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x4a
		}
	}
	if len(m.CloserPeers) > 0 {
		for iNdEx := len(m.CloserPeers) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.CloserPeers[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintDht(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x42
		}
	}
	if m.Record != nil {
		{
			size, err := m.Record.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintDht(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintDht(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0x12
	}
	if m.Type != 0 {
		i = encodeVarintDht(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Message_Peer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message_Peer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message_Peer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Connection != 0 {
		i = encodeVarintDht(dAtA, i, uint64(m.Connection))
		i--
		dAtA[i] = 0x18
	}
	if len(m.Addrs) > 0 {
		for iNdEx := len(m.Addrs) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Addrs[iNdEx])
			copy(dAtA[i:], m.Addrs[iNdEx])
			i = encodeVarintDht(dAtA, i, uint64(len(m.Addrs[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	{
		size := m.Id.Size()
		i -= size
		if _, err := m.Id.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintDht(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Message_ClosestPeers) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message_ClosestPeers) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message_ClosestPeers) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Mode != 0 {
		i = encodeVarintDht(dAtA, i, uint64(m.Mode))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Self) > 0 {
		i -= len(m.Self)
		copy(dAtA[i:], m.Self)
		i = encodeVarintDht(dAtA, i, uint64(len(m.Self)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintDht(dAtA []byte, offset int, v uint64) int {
	offset -= sovDht(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Message) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != 0 {
		n += 1 + sovDht(uint64(m.Type))
	}
	l = len(m.Key)
	if l > 0 {
		n += 1 + l + sovDht(uint64(l))
	}
	if m.Record != nil {
		l = m.Record.Size()
		n += 1 + l + sovDht(uint64(l))
	}
	if len(m.CloserPeers) > 0 {
		for _, e := range m.CloserPeers {
			l = e.Size()
			n += 1 + l + sovDht(uint64(l))
		}
	}
	if len(m.ProviderPeers) > 0 {
		for _, e := range m.ProviderPeers {
			l = e.Size()
			n += 1 + l + sovDht(uint64(l))
		}
	}
	if m.ClusterLevelRaw != 0 {
		n += 1 + sovDht(uint64(m.ClusterLevelRaw))
	}
	if m.ClosestPeers != nil {
		l = m.ClosestPeers.Size()
		n += 1 + l + sovDht(uint64(l))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovDht(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Message_Peer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Id.Size()
	n += 1 + l + sovDht(uint64(l))
	if len(m.Addrs) > 0 {
		for _, b := range m.Addrs {
			l = len(b)
			n += 1 + l + sovDht(uint64(l))
		}
	}
	if m.Connection != 0 {
		n += 1 + sovDht(uint64(m.Connection))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Message_ClosestPeers) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Self)
	if l > 0 {
		n += 1 + l + sovDht(uint64(l))
	}
	if m.Mode != 0 {
		n += 1 + sovDht(uint64(m.Mode))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovDht(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDht(x uint64) (n int) {
	return sovDht(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Message) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDht
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Message: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Message: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= Message_MessageType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = append(m.Key[:0], dAtA[iNdEx:postIndex]...)
			if m.Key == nil {
				m.Key = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Record", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Record == nil {
				m.Record = &pb.Record{}
			}
			if err := m.Record.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CloserPeers", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CloserPeers = append(m.CloserPeers, Message_Peer{})
			if err := m.CloserPeers[len(m.CloserPeers)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProviderPeers", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProviderPeers = append(m.ProviderPeers, Message_Peer{})
			if err := m.ProviderPeers[len(m.ProviderPeers)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterLevelRaw", wireType)
			}
			m.ClusterLevelRaw = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ClusterLevelRaw |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClosestPeers", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ClosestPeers == nil {
				m.ClosestPeers = &Message_ClosestPeers{}
			}
			if err := m.ClosestPeers.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = append(m.Value[:0], dAtA[iNdEx:postIndex]...)
			if m.Value == nil {
				m.Value = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDht(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDht
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Message_Peer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDht
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Peer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Peer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Id.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Addrs", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Addrs = append(m.Addrs, make([]byte, postIndex-iNdEx))
			copy(m.Addrs[len(m.Addrs)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Connection", wireType)
			}
			m.Connection = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Connection |= Message_ConnectionType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipDht(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDht
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Message_ClosestPeers) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDht
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ClosestPeers: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ClosestPeers: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Self", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthDht
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthDht
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Self = append(m.Self[:0], dAtA[iNdEx:postIndex]...)
			if m.Self == nil {
				m.Self = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Mode", wireType)
			}
			m.Mode = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDht
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Mode |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipDht(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDht
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipDht(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDht
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDht
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDht
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthDht
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDht
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDht
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDht        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDht          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDht = fmt.Errorf("proto: unexpected end of group")
)
