package decoder

import (
	"context"
	"log"
	"mempool-sniper/pkg/types"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Transaction 交易包装类型
// 由于pkg/types包导入问题，这里定义本地类型

// Decoder 交易解码器
type Decoder struct {
	mu        sync.RWMutex
	processed int64
	filtered  int64
	decoded   int64
}

// NewDecoder 创建新的解码器
func NewDecoder() *Decoder {
	return &Decoder{
		processed: 0,
		filtered:  0,
		decoded:   0,
	}
}

// StartWorkerPool 启动解码器工作池
func (d *Decoder) StartWorkerPool(ctx context.Context, txChan <-chan *types.Transaction, workerCount int) {
	log.Printf("🔍 启动解码器工作池，工作线程数: %d", workerCount)

	for i := 0; i < workerCount; i++ {
		go d.worker(ctx, txChan, i)
	}
}

// worker 解码器工作线程
func (d *Decoder) worker(ctx context.Context, txChan <-chan *types.Transaction, workerID int) {
	log.Printf("👷 解码器工作线程 %d 启动", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 解码器工作线程 %d 停止", workerID)
			return
		case tx := <-txChan:
			if tx == nil {
				continue
			}

			d.mu.Lock()
			d.processed++
			d.mu.Unlock()

			// 解码交易
			decodedTx := d.decodeTransaction(tx)
			if decodedTx != nil {
				// 这里可以添加进一步的处理逻辑
				// 例如发送到模拟器或记录日志
				log.Printf("✅ 工作线程 %d 解码成功: %s -> %s",
					workerID, tx.Hash.Hex(), decodedTx.Method)
			}
		}
	}
}

// decodeTransaction 解码交易
func (d *Decoder) decodeTransaction(tx *types.Transaction) *types.DecodedTransaction {
	if tx.To == nil {
		// 合约创建交易，跳过
		d.mu.Lock()
		d.filtered++
		d.mu.Unlock()
		return nil
	}

	// 检查是否支持该合约
	if !IsSupportedContract(*tx.To) {
		d.mu.Lock()
		d.filtered++
		d.mu.Unlock()
		return nil
	}

	// 检查交易数据长度
	if len(tx.Data) < 4 {
		d.mu.Lock()
		d.filtered++
		d.mu.Unlock()
		return nil
	}

	// 提取方法ID
	methodID := tx.Data[:4]

	// 检查是否是交换方法
	if !IsSwapMethod(methodID) {
		d.mu.Lock()
		d.filtered++
		d.mu.Unlock()
		return nil
	}

	// 构建解码后的交易信息
	decodedTx := &types.DecodedTransaction{
		Transaction:    tx,
		Method:         GetMethodName(methodID),
		MethodID:       methodID,
		TargetContract: *tx.To,
		IsSwap:         true,
	}

	// 根据方法类型设置交换方向
	switch decodedTx.Method {
	case "swapExactETHForTokens":
		decodedTx.SwapDirection = "buy"
		decodedTx.TokenIn = common.HexToAddress("0x0000000000000000000000000000000000000000") // ETH
	case "swapExactTokensForETH":
		decodedTx.SwapDirection = "sell"
		decodedTx.TokenOut = common.HexToAddress("0x0000000000000000000000000000000000000000") // ETH
	case "swapExactTokensForTokens":
		decodedTx.SwapDirection = "swap"
	}

	// 解析交易参数（简化版）
	d.parseTransactionParameters(decodedTx)

	d.mu.Lock()
	d.decoded++
	d.mu.Unlock()

	return decodedTx
}

// parseTransactionParameters 解析交易参数
func (d *Decoder) parseTransactionParameters(decodedTx *types.DecodedTransaction) {
	// 这里实现具体的参数解析逻辑
	// 由于ABI解析比较复杂，这里先实现简化版本

	data := decodedTx.Transaction.Data

	// 根据方法类型解析不同的参数
	switch decodedTx.Method {
	case "swapExactETHForTokens":
		// swapExactETHForTokens(uint amountOutMin, address[] path, address to, uint deadline)
		if len(data) >= 4+32*4 {
			// 解析amountOutMin (第1个参数)
			if len(data) >= 4+32 {
				amountOutMin := decodedTx.Transaction.Value // 使用交易价值作为输入金额
				decodedTx.AmountIn = amountOutMin
			}

			// 这里可以继续解析其他参数
			// 实际项目中需要使用完整的ABI解析
		}

	case "swapExactTokensForETH":
		// swapExactTokensForETH(uint amountIn, uint amountOutMin, address[] path, address to, uint deadline)
		if len(data) >= 4+32*5 {
			// 解析amountIn (第1个参数)
			if len(data) >= 4+32 {
				// 这里需要从calldata中提取amountIn
				// 简化处理：使用交易价值
				decodedTx.AmountIn = decodedTx.Transaction.Value
			}
		}

	case "swapExactTokensForTokens":
		// swapExactTokensForTokens(uint amountIn, uint amountOutMin, address[] path, address to, uint deadline)
		if len(data) >= 4+32*5 {
			// 解析amountIn (第1个参数)
			if len(data) >= 4+32 {
				// 简化处理
				decodedTx.AmountIn = decodedTx.Transaction.Value
			}
		}
	}
}

// GetStats 获取统计信息
func (d *Decoder) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return map[string]interface{}{
		"processed": d.processed,
		"filtered":  d.filtered,
		"decoded":   d.decoded,
		"success_rate": func() float64 {
			if d.processed == 0 {
				return 0
			}
			return float64(d.decoded) / float64(d.processed) * 100
		}(),
	}
}

// FilterTransaction 过滤交易（公开方法，可供外部调用）

// IsSupportedContract 检查是否支持该合约
func IsSupportedContract(address common.Address) bool {
	_, exists := SupportedDEX[address]
	return exists
}

// IsSwapMethod 检查是否是交换方法
func IsSwapMethod(methodID []byte) bool {
	for _, supportedMethod := range SupportedSwapMethods {
		if len(methodID) >= 4 && len(supportedMethod) >= 4 {
			if string(methodID[:4]) == string(supportedMethod[:4]) {
				return true
			}
		}
	}
	return false
}

// GetMethodName 根据方法ID获取方法名称
func GetMethodName(methodID []byte) string {
	for name, id := range SupportedSwapMethods {
		if len(methodID) >= 4 && len(id) >= 4 {
			if string(methodID[:4]) == string(id[:4]) {
				return name
			}
		}
	}
	return "unknown"
}

// 预定义的合约地址和方法签名
var (
	// 常见DEX路由器地址
	UniswapV2Router = common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")
	UniswapV3Router = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
	SushiSwapRouter = common.HexToAddress("0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F")

	// 常见交换方法签名
	MethodSwapExactETHForTokens    = []byte{0x7f, 0xf3, 0x6a, 0xb5} // swapExactETHForTokens
	MethodSwapExactTokensForETH    = []byte{0x18, 0xcb, 0xaf, 0x05} // swapExactTokensForETH
	MethodSwapExactTokensForTokens = []byte{0x38, 0xed, 0x17, 0x39} // swapExactTokensForTokens

	// 支持的DEX列表
	SupportedDEX = map[common.Address]string{
		UniswapV2Router: "Uniswap V2",
		UniswapV3Router: "Uniswap V3",
		SushiSwapRouter: "SushiSwap",
	}

	// 支持的交换方法
	SupportedSwapMethods = map[string][]byte{
		"swapExactETHForTokens":    MethodSwapExactETHForTokens,
		"swapExactTokensForETH":    MethodSwapExactTokensForETH,
		"swapExactTokensForTokens": MethodSwapExactTokensForTokens,
	}
)

// FilterTransaction 过滤交易（公开方法，可供外部调用）
func (d *Decoder) FilterTransaction(tx *types.Transaction) bool {
	if tx.To == nil {
		return false
	}

	if !IsSupportedContract(*tx.To) {
		return false
	}

	if len(tx.Data) < 4 {
		return false
	}

	methodID := tx.Data[:4]
	return IsSwapMethod(methodID)
}

// DecodeTransaction 解码交易（公开方法，可供外部调用）
func (d *Decoder) DecodeTransaction(tx *types.Transaction) *types.DecodedTransaction {
	return d.decodeTransaction(tx)
}
