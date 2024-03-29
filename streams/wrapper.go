package streams

import (
	"io"
	"net"
	"runtime/debug"
	"time"

	"github.com/bpfs/dep2p/utils"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio"
	"github.com/sirupsen/logrus"
)

// 表示消息头的长度
const (
	messageHeaderLen = 17
	// MaxBlockSize     = 20000000 // 20M

)

var (
	messageHeader = headerSafe([]byte("/protobuf/msgio"))
	// 最大重试次数
	maxRetries = 3
	// 基础超时时间
	baseTimeout = 5 * time.Second
	// 最大消息大小
	MaxBlockSize = 1 << 25 // 32MB
)

// ReadStream 从流中读取消息，带指数退避重试
// 特别是考虑到避免资源的过度消耗和处理网络故障情况，引入指数退避重试策略，并设置合理的重试次数上限。
// 重要的改进点：
// 指数退避：每次重试的超时时间都会增加，减少在网络不稳定时的重试频率，减轻服务器压力。
// 有限的重试次数：通过maxRetries限制重试次数，防止在遇到持续的网络问题时无限重试。
// 清晰的错误处理：根据错误类型决定是否重试。例如，只有在遇到网络超时或其他指定的网络错误时才进行重试。
func ReadStream(stream network.Stream) ([]byte, error) {
	var (
		header [messageHeaderLen]byte
		err    error
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 使用指数退避设置读取操作的超时时间
		timeout := baseTimeout * time.Duration(1<<attempt)
		stream.SetReadDeadline(time.Now().Add(timeout))

		// 尝试读取消息头部
		if _, err = io.ReadFull(stream, header[:]); err != nil {
			// 如果错误是网络超时或暂时性的，考虑重试
			if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
				logrus.Errorf("[%s]: 尝试%d次读取头部失败，网络错误: %v", utils.WhereAmI(), attempt+1, err)
				continue
			}
			// 对于非网络超时、非暂时性错误，直接退出
			break
		}

		// 如果头部读取成功，退出重试循环
		break
	}

	if err != nil {
		logrus.Errorf("[%s]: 经过%d次重试后读取失败: %v", utils.WhereAmI(), maxRetries, err)
		return nil, err
	}

	// 成功读取头部后，继续读取消息体，此时重置超时以避免中断正常读取
	stream.SetReadDeadline(time.Time{})
	reader := msgio.NewReaderSize(stream, MaxBlockSize)
	msg, err := reader.ReadMsg()
	if err != nil {
		logrus.Errorf("[%s]: 读取消息体失败: %v", utils.WhereAmI(), err)
		return nil, err
	}
	defer reader.ReleaseMsg(msg)

	return msg, nil
}

// WriteStream 将消息写入流
// 采用类似于ReadStream的优化策略来增强其鲁棒性，尤其是在面对网络问题时。
// 重点将包括设置写操作的超时时间和对潜在的写操作失败进行适当的错误处理。
// 考虑到WriteStream的操作通常比读操作更简单（通常是一次性写入而非多次读取），通过设置整体的写超时来简化流程。
func WriteStream(msg []byte, stream network.Stream) error {
	// 设置写操作的超时时间
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		logrus.Errorf("[%s]: 设置写入超时失败: %v", utils.WhereAmI(), err)
		return err
	}

	// 先写入消息头
	_, err := stream.Write(messageHeader)
	if err != nil {
		logrus.Errorf("[%s]: 写入头部失败: %v", utils.WhereAmI(), err)
		return err
	}

	// 使用msgio包装流，以便于写入长度前缀和消息体
	writer := msgio.NewWriterWithPool(stream, pool.GlobalPool)
	if err = writer.WriteMsg(msg); err != nil {
		logrus.Errorf("[%s]: 写入消息失败: %v", utils.WhereAmI(), err)
		return err
	}

	// 写入操作成功后，清除写超时设置
	if err := stream.SetWriteDeadline(time.Time{}); err != nil {
		logrus.Errorf("[%s]: 清除写入超时设置失败: %v", utils.WhereAmI(), err)
		// 此处不返回错误，因为主要写入操作已经成功完成
	}

	return nil
}

// CloseStream 写入后关闭流，并等待EOF。
func CloseStream(stream network.Stream) error {
	if stream == nil {
		return nil
	}

	// 关闭写方向。如果流是全双工的，这会发送EOF给读方，而不会关闭整个流。
	if err := stream.CloseWrite(); err != nil {
		logrus.Errorf("[%s]: 关闭写方向失败: %v", utils.WhereAmI(), err)
		return err
	}

	// 关闭读方向。如果不需要读取EOF，也可以省略这一步。
	if err := stream.CloseRead(); err != nil {
		logrus.Errorf("[%s]: 关闭读方向失败: %v", utils.WhereAmI(), err)
		return err
	}

	go func() {
		// 设置超时，以防AwaitEOF卡住
		timer := time.NewTimer(EOFTimeout)
		defer timer.Stop()

		done := make(chan error, 1)
		go func() {
			// AwaitEOF等待给定流上的EOF，如果失败则返回错误。
			err := AwaitEOF(stream)
			done <- err
		}()

		select {
		case <-timer.C:
			// logrus.Errorf("[%s]: 等待EOF超时", utils.WhereAmI())
			// 超时后重置流，确保资源被释放
			if err := stream.Reset(); err != nil {
				logrus.Errorf("[%s]: 重置流失败: %v", utils.WhereAmI(), err)
			}
		case err := <-done:
			if err != nil {
				// logrus.Errorf("[%s]: 等待EOF时出错: %v", utils.WhereAmI(), err)
				// 有错误时记录，但通常不需要额外操作
			}
		}
	}()

	return nil
}

// HandlerWithClose 用关闭流和从恐慌中恢复来包装处理程序
func HandlerWithClose(f network.StreamHandler) network.StreamHandler {
	return func(stream network.Stream) {
		defer func() {
			// 从panic中恢复并记录错误信息
			if r := recover(); r != nil {
				logrus.Errorf("[%s]处理流恐慌错误: %v", utils.WhereAmI(), r)
				// 打印堆栈信息以便调试
				logrus.Errorf("Panic stack trace: %s", string(debug.Stack()))
				// 尝试重置stream以确保资源被释放
				if err := stream.Reset(); err != nil {
					logrus.Errorf("[%s]: 尝试重置stream失败: %v", utils.WhereAmI(), err)
				}
			}
		}()

		// 调用原始的stream处理函数
		f(stream)

		// 尝试优雅地关闭stream
		if err := CloseStream(stream); err != nil {
			logrus.Errorf("[%s]: 关闭stream时发生错误: %v", utils.WhereAmI(), err)
		}
	}
}

// ReadStream 从流中读取消息
// func ReadStream(stream network.Stream) ([]byte, error) {
// 	var header [messageHeaderLen]byte

// 	_, err := io.ReadFull(stream, header[:])
// 	if err != nil || !bytes.Equal(header[:], messageHeader) {
// 		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
// 		logrus.Error("ReadStream", "pid", stream.Conn().RemotePeer().String(), "protocolID", stream.Protocol(), "read header err", err)
// 		return nil, err
// 	}

// 	reader := msgio.NewReaderSize(stream, MaxBlockSize)
// 	msg, err := reader.ReadMsg()
// 	if err != nil {
// 		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
// 		logrus.Error("ReadStream", "pid", stream.Conn().RemotePeer().String(), "protocolID", stream.Protocol(), "read msg err", err)
// 		return nil, err
// 	}

// 	defer reader.ReleaseMsg(msg)

// 	return msg, nil
// }

// WriteStream 将消息写入流
// func WriteStream(msg []byte, stream network.Stream) error {
// 	_, err := stream.Write(messageHeader)
// 	if err != nil {
// 		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
// 		logrus.Error("WriteStream", "pid", stream.Conn().RemotePeer().String(), "protocolID", stream.Protocol(), "write header err", err)
// 		return err
// 	}

// 	// NewWriter 包装了一个 io.Writer 和一个 msgio 框架的作者。 msgio.Writer 将写入每条消息的长度前缀。
// 	// writer := msgio.NewWriter(stream)
// 	// NewWriterWithPool 与 NewWriter 相同，但允许用户传递自定义缓冲池。
// 	writer := msgio.NewWriterWithPool(stream, pool.GlobalPool)

// 	// WriteMsg 将消息写入传入的缓冲区。
// 	if err = writer.WriteMsg(msg); err != nil {
// 		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
// 		logrus.Error("WriteStream", "pid", stream.Conn().RemotePeer().String(), "protocolID", stream.Protocol(), "write msg err", err)
// 		return err
// 	}

// 	return nil
// }

// CloseStream 写入后关闭流，并等待 EOF。
// func CloseStream(stream network.Stream) {
// 	if stream == nil {
// 		return
// 	}

// 	// _ = stream.CloseWrite()
// 	// _ = stream.CloseRead()

// 	go func() {
// 		// AwaitEOF 等待给定流上的 EOF，如果失败则返回错误。 它最多等待 EOFTimeout（默认为 1 分钟），然后重置流。
// 		err := AwaitEOF(stream)
// 		if err != nil {
// 			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
// 			// 只是记录它，因为这无关紧要
// 			logrus.Debug("CloseStream", "err", err, "protocol ID", stream.Protocol())
// 		}
// 	}()
// }

// HandlerWithClose 用关闭流和从恐慌中恢复来包装处理程序
//
//	func HandlerWithClose(f network.StreamHandler) network.StreamHandler {
//		return func(stream network.Stream) {
//			defer func() {
//				// recover 内置函数允许程序管理恐慌 goroutine 的行为。
//				// 在延迟函数（但不是它调用的任何函数）内执行恢复调用，通过恢复正常执行来停止恐慌序列，并检索传递给恐慌调用的错误值。
//				// 如果在延迟函数之外调用 recover，它不会停止恐慌序列。
//				// 在这种情况下，或者当 goroutine 没有 panic 时，或者如果提供给 panic 的参数是 nil，recover 返回 nil。
//				// 因此 recover 的返回值报告了 goroutine 是否 panicing。
//				if r := recover(); r != nil {
//					logrus.Errorf("[%s]处理流恐慌错误:%v", utils.WhereAmI(), r)
//					fmt.Println(string(panicTrace(4)))
//					// 关闭流的两端。 用它来告诉远端挂断电话并离开。
//					_ = stream.Reset()
//				}
//			}()
//			f(stream)
//			// 写入后关闭流，并等待 EOF。
//			CloseStream(stream)
//		}
//	}
//

// HandlerWithWrite 通过写入、关闭流和从恐慌中恢复来包装处理程序
func HandlerWithWrite(f func(request *RequestMessage) error) network.StreamHandler {
	return func(stream network.Stream) {
		var req RequestMessage
		if err := f(&req); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return
		}

		// 序列化请求
		requestByte, err := req.Marshal()
		if err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return
		}

		// WriteStream 将消息写入流。
		if err := WriteStream(requestByte, stream); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return
		}
	}
}

// HandlerWithRead 用读取、关闭流和从恐慌中恢复来包装处理程序
func HandlerWithRead(f func(request *RequestMessage)) network.StreamHandler {
	return func(stream network.Stream) {

		var req RequestMessage

		// ReadStream 从流中读取消息。
		requestByte, err := ReadStream(stream)
		if err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return
		}
		if err := req.Unmarshal(requestByte); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return
		}

		f(&req)
	}
}

// HandlerWithRW 用读取、写入、关闭流和从恐慌中恢复来包装处理程序
// func HandlerWithRW(f func(request *RequestMessage, response *ResponseMessage) error) network.StreamHandler {
// 	// 返回一个网络流处理器函数
// 	return func(stream network.Stream) {

// 		var req RequestMessage
// 		var res ResponseMessage
// 		res.Code = 201
// 		res.Message = &Message{
// 			Sender: stream.Conn().LocalPeer().String(), // 发送方ID
// 		}
// 		//res.Message.Sender = stream.Conn().LocalPeer().String()

// 		// 从流中读取消息
// 		requestByte, err := ReadStream(stream)
// 		if err != nil {
// 			res.Msg = "请求无法处理" // 响应消息

// 		} else if len(requestByte) == 0 {
// 			return
// 		} else if len(requestByte) > 0 {
// 			if err := req.Unmarshal(requestByte); err != nil {
// 				// 处理请求解析错误
// 				res.Msg = "请求解析错误"
// 			} else {
// 				// 调用处理函数处理请求并获取响应
// 				if err := f(&req, &res); err != nil {
// 					res.Msg = err.Error()
// 				}
// 			}
// 		}

// 		responseByte, err := res.Marshal()
// 		if err != nil {
// 			return
// 		}
// 		// 将响应消息写入流中
// 		if err := WriteStream(responseByte, stream); err != nil {
// 			return
// 		}
// 	}
// }

// HandlerWithRW 用于读取、写入、关闭流以及从恐慌中恢复，来包装处理程序。
// 处理程序 f 现在接收 RequestMessage 和 ResponseMessage，允许直接在函数内部定义成功或错误的响应。
func HandlerWithRW(f func(request *RequestMessage, response *ResponseMessage) (int32, string)) network.StreamHandler {
	return func(stream network.Stream) {
		var req RequestMessage
		var res ResponseMessage

		// 从流中读取请求消息
		requestByte, err := ReadStream(stream)
		if err != nil || len(requestByte) == 0 {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			logrus.Errorf("读取请求失败: %v", err)
			SendErrorResponse(stream, 500, "读取请求失败")
			return
		}

		// 解析请求消息
		if err := req.Unmarshal(requestByte); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			logrus.Errorf("请求解析错误: %v", err)
			SendErrorResponse(stream, 400, "请求解析错误")
			return
		}

		// 调用处理函数处理请求，并根据返回的错误码和消息设置响应
		code, msg := f(&req, &res)

		// 设置响应码和消息
		res.Code = code
		res.Msg = msg

		// 将处理后的响应消息编码并写入流中
		responseByte, err := res.Marshal()
		if err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			logrus.Errorf("响应编码失败: %v", err)
			SendErrorResponse(stream, 500, "响应编码失败")
			return
		}

		if err := WriteStream(responseByte, stream); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			logrus.Errorf("写入响应消息失败: %v", err)
			// 这里不再返回，因为已经是发送响应的步骤
		}
	}
}

// SendErrorResponse 是一个辅助函数，用于向流中发送错误响应。
func SendErrorResponse(stream network.Stream, code int32, msg string) {
	res := ResponseMessage{
		Code:    code,
		Message: &Message{Sender: stream.Conn().LocalPeer().String()},
		Msg:     msg,
	}
	responseByte, _ := res.Marshal()
	_ = WriteStream(responseByte, stream) // 这里忽略错误处理，因为已在错误路径中
}

// GetStatusDescription 获取状态码对应的描述信息
func GetStatusDescription(code int32) string {
	if desc, ok := StatusCodeDescriptions[code]; ok {
		return desc
	}
	return "未知状态码"
}

// 示例：使用状态码
// func ExampleHandler() {
// 	code := int32(200)                // 假设这是处理函数中决定的响应码
// 	msg := GetStatusDescription(code) // 根据状态码获取描述信息
// 	// 在这里构建响应消息并发送...
// }

/**
分类设计
基础与通用（100-299）

HTTP标准状态码，用于表示通用的成功、错误等状态。
区块链核心操作（2000-2999）

包含交易处理、区块验证、共识机制等区块链特有的操作状态。
数据与数据库操作（3000-3999）

涉及数据的CRUD操作、查询优化、事务处理等数据库相关状态。
文件与去中心化存储（4000-4999）

关于文件的上传下载、分片、加密、去中心化存储策略等状态。
网络与P2P通信（5000-5999）

包括节点发现、数据同步、网络延迟、传输加密等P2P网络特有的问题。
安全性与隐私保护（6000-6999）

涉及加密、解密、签名、验签、隐私数据保护等安全操作。
钱包与资产管理（7000-7999）

钱包创建、资产转账、助记词恢复、密钥管理等钱包相关操作。
智能合约与链上应用（8000-8999）

智能合约部署、调用、执行结果、链上应用交互等状态。
系统与资源管理（9000-9999）

系统维护、资源分配、性能优化、状态监控等系统级操作。
*/

// 常用状态码及描述
var StatusCodeDescriptions = map[int32]string{
	// 通用状态码
	200: "成功",      // 请求或操作成功处理
	204: "无内容",     // 请求成功，但响应体中无内容
	400: "请求参数错误",  // 请求中存在语法问题或无法满足请求
	401: "未授权",     // 请求缺少有效的认证信息
	403: "禁止访问",    // 服务器拒绝执行请求
	404: "资源未找到",   // 请求的资源未在服务器上找到
	408: "请求超时",    // 服务器等待请求时超时
	500: "内部服务器错误", // 服务器遇到错误，无法完成请求
	501: "未实现",     // 服务器不支持请求的功能
	502: "网关错误",    // 作为网关或代理工作的服务器从上游服务器收到无效响应
	503: "服务不可用",   // 由于临时的服务器维护或过载，服务器当前无法处理请求

	// 基础与通用状态码（1000-1099）
	1000: "操作成功",  // 请求或操作成功被执行并处理。
	1001: "操作待处理", // 操作已接收，正在等待处理。
	1002: "内容为空",  // 请求成功执行，但未返回任何内容。
	// 请求与参数相关（1100-1199）
	1100: "请求格式错误",  // 请求的格式不符合预期标准，无法被解析。
	1101: "参数缺失或无效", // 请求中缺少必要的参数，或参数格式不正确。
	// 认证与权限（1200-1299）
	1200: "认证失败", // 提供的认证信息无效或认证过程失败。
	1201: "权限不足", // 当前用户权限不足，无法执行请求的操作。
	// 资源与数据（1300-1399）
	1300: "资源不存在",  // 请求的资源或数据未在系统中找到。
	1301: "数据验证失败", // 提供的数据未通过验证过程。
	// 操作与执行（1400-1499）
	1400: "操作被拒绝", // 请求的操作因特定原因被系统拒绝。
	1401: "操作超时",  // 请求的操作未在允许的时间内完成。
	// 服务与依赖（1500-1599）
	1500: "依赖服务不可用", // 请求操作所依赖的外部服务当前不可用。
	1501: "服务暂时不可用", // 请求的服务因维护或过载暂时不可用。
	// 错误与异常（1600-1699）
	1600: "未知错误",  // 遇到未预期的错误，具体原因可能未知。
	1601: "操作未实现", // 请求的操作或方法在当前上下文中未实现。
	// 其他（1700-1799）
	1700: "操作待确认",  // 操作已接受，正在等待用户或系统的进一步确认。
	1701: "功能即将推出", // 用户请求的功能已计划实现，但当前版本尚未支持。

	// 交易处理状态码（2100-2199）
	2100: "交易提交成功", // 交易已成功接收，并等待网络处理。
	2101: "交易验证失败", // 交易未通过网络规则验证。
	2102: "交易执行失败", // 交易执行中遇到问题，无法完成。
	2103: "交易已存在",  // 提交的交易已在区块链网络中存在。
	2104: "交易池已满",  // 网络的交易池容量已达上限。
	2105: "交易已过期",  // 交易因长时间未被处理而过期。
	// 区块处理状态码（2200-2299）
	2200: "区块验证失败", // 提交的区块未通过验证。
	2201: "区块添加成功", // 新区块已成功添加至区块链。
	2202: "区块已存在",  // 提交的区块已在区块链中存在。
	2203: "区块丢失",   // 未找到预期中应存在的区块。
	2204: "区块冲突",   // 新提交的区块与现有区块链信息发生冲突。
	// 共识机制状态码（2300-2399）
	2300: "共识达成",   // 节点间就某项数据或决策成功达成一致。
	2301: "共识失败",   // 节点间未能就某项数据或决策达成一致。
	2302: "共识过程错误", // 在尝试达成共识的过程中遇到错误。
	// 区块链网络与系统配置错误（2400-2499）
	2400: "网络配置错误", // 区块链网络配置存在问题，影响正常通信。
	2401: "系统配置错误", // 区块链系统配置错误，影响运行。
	// 区块链数据管理与同步（2500-2599）
	2500: "数据请求成功", // 请求的链上数据成功返回。
	2501: "数据不存在",  // 请求的数据在区块链中未找到。
	2502: "数据冲突",   // 提交的数据与现有链上数据发生冲突。
	// 区块链资源管理（2600-2699）
	2600: "资源请求成功", // 对链上资源的请求成功处理。
	2601: "资源不足",   // 区块链系统资源不足，无法完成请求。
	2602: "资源访问冲突", // 对同一资源的并发访问导致冲突。

	// 数据查询操作（3100-3199）
	3100: "数据查询成功", // 数据查询操作成功，返回请求的数据。
	3101: "数据不存在",  // 请求的数据在数据库中未找到。
	3102: "查询参数错误", // 数据查询请求中包含无效的查询参数。
	// 数据插入操作（3200-3299）
	3200: "数据插入成功", // 新数据成功插入数据库。
	3201: "数据已存在",  // 尝试插入的数据在数据库中已存在。
	3202: "插入参数错误", // 数据插入请求中包含无效的参数。
	// 数据更新操作（3300-3399）
	3300: "数据更新成功",  // 数据更新操作成功完成。
	3301: "更新目标不存在", // 尝试更新的数据在数据库中不存在。
	3302: "更新冲突",    // 数据更新操作引发的冲突，可能由并发控制引起。
	// 数据删除操作（3400-3499）
	3400: "数据删除成功",  // 数据删除成功完成。
	3401: "删除目标不存在", // 尝试删除的数据在数据库中不存在。
	3402: "删除操作受限",  // 数据删除操作受到限制或保护，无法完成。
	// 事务处理（3500-3599）
	3500: "事务提交成功", // 数据库事务成功提交。
	3501: "事务回滚",   // 数据库事务回滚，可能由执行错误或并发冲突引起。
	3502: "事务处理异常", // 在事务处理过程中遇到异常。
	// 数据同步与备份（3600-3699）
	3600: "数据同步成功",  // 数据同步操作成功完成。
	3601: "数据备份成功",  // 数据备份操作成功完成。
	3602: "同步目标不可达", // 数据同步的目标数据库或服务不可达。
	3603: "备份失败",    // 数据备份操作失败，可能由外部存储问题引起。
	// 并发控制与锁定（3700-3799）
	3700: "锁定成功", // 成功获得数据或资源的锁定。
	3701: "锁定冲突", // 数据或资源锁定操作引发的冲突。
	3702: "锁定超时", // 尝试锁定数据或资源时操作超时。

	// 文件基本操作（4100-4199）
	4100: "文件上传成功", // 文件成功上传至存储系统。
	4101: "文件下载成功", // 文件成功从存储系统下载。
	4102: "文件删除成功", // 文件成功从存储系统删除。
	4103: "文件更新成功", // 文件内容或元数据成功更新。
	4104: "文件不存在",  // 请求的文件在存储系统中未找到。
	4105: "文件访问受限", // 对文件的访问因权限不足被拒绝。
	// 存储空间管理（4200-4299）
	4200: "存储空间配置成功", // 存储空间成功配置或扩展。
	4201: "存储空间不足",   // 存储系统空间不足，无法完成操作。
	4202: "存储空间释放成功", // 存储空间成功释放或减小。
	4203: "存储空间分配失败", // 存储空间分配或扩展操作失败。
	// 去中心化存储特性（4300-4399）
	4300: "数据分片成功",     // 数据成功分片并存储于去中心化网络。
	4301: "数据重组成功",     // 分片数据成功重组恢复原始文件。
	4302: "数据分片验证失败",   // 数据分片在验证过程中未能通过。
	4303: "去中心化网络不可达",  // 去中心化存储网络当前不可访问。
	4304: "去中心化存储冗余不足", // 存储数据的冗余度不足，存在丢失风险。
	// 文件加密与安全（4400-4499）
	4400: "文件加密成功", // 文件内容成功加密。
	4401: "文件解密成功", // 文件内容成功解密。
	4402: "文件加密失败", // 文件加密操作失败。
	4403: "文件解密失败", // 文件解密操作失败。
	4404: "安全策略违规", // 文件操作违反了安全策略。
	// 文件版本控制与历史（4500-4599）
	4500: "文件版本创建成功",  // 新的文件版本成功创建。
	4501: "文件版本恢复成功",  // 文件成功恢复至指定版本。
	4502: "文件历史记录不可用", // 请求的文件历史记录不可用或不存在。
	4503: "文件版本冲突",    // 文件版本更新操作引发的版本冲突。

	// 网络连接管理（5100-5199）
	5100: "网络连接成功", // 成功建立网络连接。
	5101: "网络连接断开", // 网络连接被断开。
	5102: "网络连接超时", // 尝试建立网络连接时超时。
	5103: "网络配置错误", // 网络配置不正确导致连接失败。
	// 数据传输与同步（5200-5299）
	5200: "数据传输成功", // 数据成功发送或接收。
	5201: "数据传输失败", // 数据在传输过程中失败。
	5202: "数据同步完成", // 数据同步操作成功完成。
	5203: "数据同步失败", // 数据同步操作失败。
	// P2P网络特性（5300-5399）
	5300: "节点发现成功", // 成功发现新的P2P网络节点。
	5301: "节点连接建立", // 与P2P网络节点成功建立连接。
	5302: "节点连接失败", // 尝试与P2P网络节点建立连接失败。
	5303: "网络拓扑更新", // P2P网络拓扑结构成功更新。
	5304: "网络分区检测", // 检测到P2P网络可能存在分区问题。
	// 网络安全与隐私（5400-5499）
	5400: "加密通信成功", // 加密通信协议成功建立。
	5401: "加密通信失败", // 加密通信协议建立失败。
	5402: "身份验证成功", // 节点身份验证成功。
	5403: "身份验证失败", // 节点身份验证失败。
	5404: "匿名路由成功", // 成功通过匿名路由发送数据。
	5405: "匿名路由失败", // 匿名路由过程中出现错误。
	// 性能与可靠性（5500-5599）
	5500: "网络延迟测量",  // 成功完成网络延迟测量。
	5501: "网络可靠性评估", // 完成网络可靠性评估。
	5502: "网络拥塞检测",  // 检测到网络拥塞情况。
	5503: "数据重传请求",  // 由于丢包等问题请求数据重传。
	5504: "负载均衡调整",  // 网络负载均衡策略调整完成。

	// 安全性与隐私保护
	// 认证与授权（6100-6199）
	6100: "认证成功",  // 用户认证或数据签名验证成功。
	6101: "认证失败",  // 用户认证失败，提供的认证信息无效。
	6102: "权限不足",  // 用户权限不足以进行当前操作。
	6103: "会话过期",  // 用户会话已过期，需要重新认证。
	6104: "访问被拒绝", // 用户的访问请求被系统拒绝。
	// 数据加解密（6200-6299）
	6200: "加密成功",   // 数据成功加密。
	6201: "解密成功",   // 数据成功解密。
	6202: "加密失败",   // 数据加密过程中出现错误。
	6203: "解密失败",   // 数据解密过程中出现错误。
	6204: "加密密钥失效", // 使用的加密密钥已失效或不正确。
	// 签名与验签（6300-6399）
	6300: "签名成功",   // 数据签名成功。
	6301: "验签成功",   // 数据验签成功，签名有效。
	6302: "签名失败",   // 数据签名过程中出现错误。
	6303: "验签失败",   // 数据验签失败，签名无效或数据被篡改。
	6304: "签名密钥失效", // 使用的签名密钥已失效或不正确。
	// 安全策略违规（6400-6499）
	6400: "安全策略违规", // 操作违反了系统的安全策略。
	6401: "访问受限",   // 操作因安全策略而受到限制。
	6402: "敏感数据访问", // 尝试访问受保护的敏感数据。
	6403: "安全漏洞利用", // 检测到尝试利用已知安全漏洞。
	6404: "非法入侵尝试", // 系统检测到非法入侵的尝试。
	// 其他安全问题（6500-6599）
	6500: "数据篡改",   // 检测到数据被未授权篡改。
	6501: "重放攻击",   // 检测到重放攻击尝试。
	6502: "拒绝服务攻击", // 系统正遭受拒绝服务攻击。
	6503: "跨站脚本攻击", // 检测到跨站脚本攻击尝试。
	6504: "系统安全更新", // 系统进行安全更新，可能暂时影响服务。

	// 钱包基础操作（7000-7099）
	7000: "钱包创建成功", // 成功创建了新的钱包。
	7001: "钱包访问成功", // 成功访问了钱包数据。
	7002: "钱包余额不足", // 执行操作时发现钱包的余额不足。
	7003: "钱包地址无效", // 提供的钱包地址格式不正确或不存在。
	7004: "钱包备份成功", // 成功备份了钱包。
	7005: "钱包恢复成功", // 从备份中成功恢复了钱包。
	7006: "钱包操作失败", // 在进行钱包操作时发生了未指定的错误。
	// 助记词与密码操作（7100-7199）
	7100: "助记词验证成功", // 助记词成功通过验证。
	7101: "密码验证成功",  // 密码成功通过验证。
	7102: "密码更新成功",  // 成功更新了密码。
	7103: "助记词验证失败", // 提供的助记词未能通过验证。
	7104: "密码验证失败",  // 提供的密码未能通过验证。
	7105: "密码更新失败",  // 尝试更新密码时发生了错误。
	// 密钥管理（7200-7299）
	7200: "密钥导入成功", // 成功导入了密钥。
	7201: "密钥导出成功", // 成功导出了密钥。
	7202: "密钥生成成功", // 成功生成了新的密钥。
	7203: "密钥更新成功", // 成功更新了密钥。
	7204: "密钥删除成功", // 成功删除了密钥。
	7205: "密钥导入失败", // 在导入密钥时发生了错误。
	7206: "密钥导出失败", // 在导出密钥时发生了错误。
	7207: "密钥生成失败", // 在生成新的密钥时发生了错误。
	7208: "密钥更新失败", // 在更新密钥时发生了错误。
	7209: "密钥删除失败", // 在删除密钥时发生了错误。
	// 资产管理（7300-7399）
	7300: "资产转移成功", // 资产成功从一个账户转移到另一个账户。
	7301: "资产冻结成功", // 资产成功被冻结，暂时不可用。
	7302: "资产解冻成功", // 资产成功被解冻，恢复可用。
	7303: "资产查询成功", // 成功查询了资产信息。
	7304: "资产操作失败", // 在进行资产相关操作时发生了错误。
	// 助记词与密钥恢复（7400-7499）
	7400: "助记词恢复成功", // 使用助记词成功恢复了钱包。
	7401: "密钥恢复成功",  // 使用备份密钥成功恢复了钱包访问权限。
	7402: "助记词恢复失败", // 使用助记词恢复钱包时发生了错误。
	7403: "密钥恢复失败",  // 使用备份密钥恢复钱包时发生了错误。

	// 应用操作（8000-8099）
	8000: "应用请求成功",    // 链上应用处理请求成功。
	8001: "应用层数据不一致",  // 应用层数据与链上数据存在不一致性。
	8002: "应用状态更新成功",  // 应用状态或数据更新成功完成。
	8003: "应用调用失败",    // 链上应用调用或执行失败。
	8004: "应用层服务不可用",  // 请求的链上应用服务暂时不可用。
	8005: "应用层访问权限不足", // 请求的操作因权限不足被拒绝。
	8006: "应用层请求参数错误", // 应用层请求包含无效参数或格式错误。
	8007: "应用层操作超时",   // 应用层操作未在预期时间内完成。
	8008: "应用层网络错误",   // 应用层面遇到网络通信问题。
	8009: "应用配置错误",    // 应用配置不正确或存在问题。

	// 系统基础与维护（9000-9099）
	9000: "系统启动成功",   // 系统成功启动并运行。
	9001: "系统维护中",    // 系统当前处于维护状态，部分或全部服务可能不可用。
	9002: "系统升级完成",   // 系统升级操作成功完成，功能可能有所更新。
	9003: "系统恢复成功",   // 系统从备份或异常状态成功恢复。
	9004: "维护模式开启",   // 系统进入维护模式，正常服务可能会暂时中断。
	9005: "维护模式结束",   // 系统维护结束，所有服务恢复正常运行。
	9006: "系统性能优化成功", // 对系统进行的性能优化操作成功完成。
	9007: "系统状态更新",   // 系统状态信息被更新，可能是监控或配置变更引起。
	// 资源管理与监控（9100-9199）
	9100: "资源请求成功", // 对系统资源的请求成功处理，资源被成功分配。
	9101: "资源释放成功", // 占用的系统资源成功释放。
	9102: "资源不足",   // 系统资源不足，无法满足进一步的请求。
	9103: "资源访问冲突", // 多个操作尝试同时访问或修改资源，导致冲突。
	9104: "资源过期",   // 请求的资源已经过期，可能是缓存或临时数据。
	9105: "资源锁定成功", // 资源被成功锁定，防止并发访问冲突。
	9106: "资源锁定解除", // 资源的锁定状态被成功解除。
	9107: "资源更新冲突", // 尝试更新的资源与其他操作发生冲突，需要协调。
	// 系统安全与隐私（9200-9299）
	9200: "安全检查通过", // 系统安全检查通过，没有发现安全威胁。
	9201: "隐私保护启用", // 系统隐私保护功能被成功启用，用户数据受到保护。
	9202: "访问控制更新", // 系统的访问控制列表或策略更新成功。
	9203: "安全策略违规", // 检测到操作违反了系统的安全策略。
	9204: "安全漏洞修复", // 系统中的安全漏洞被成功修复。
	// 错误与异常管理（9300-9399）
	9300: "未知错误",   // 系统遇到未预期的未知错误。
	9301: "操作失败",   // 请求的操作未能成功执行，可能是内部错误或条件不满足。
	9302: "依赖服务异常", // 依赖的外部服务或资源发生异常，影响了当前操作。
	9303: "系统资源不足", // 系统资源不足，无法完成请求的操作。
	9304: "网络连接异常", // 系统遇到网络连接问题，可能是网络不稳定或配置错误。
}
