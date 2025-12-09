package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Transaction 交易包装类型
type Transaction struct {
	Hash        common.Hash     `json:"hash"`
	RawTx       *types.Transaction `json:"raw_tx"`
	From        common.Address  `json:"from"`
	To          *common.Address `json:"to"`
	Value       *big.Int        `json:"value"`
	GasPrice    *big.Int        `json:"gas_price"`
	GasLimit    uint64          `json:"gas_limit"`
	Data        []byte          `json:"data"`
	Nonce       uint64          `json:"nonce"`
	ChainID     *big.Int        `json:"chain_id"`
	Timestamp   int64           `json:"timestamp"`
}

// DecodedTransaction 解码后的交易信息
type DecodedTransaction struct {
	Transaction     *Transaction `json:"transaction"`
	Method          string       `json:"method"`
	MethodID        []byte       `json:"method_id"`
	TargetContract  common.Address `json:"target_contract"`
	Parameters      []interface{} `json:"parameters"`
	IsSwap          bool         `json:"is_swap"`
	SwapDirection   string       `json:"swap_direction"` // "buy" or "sell"
	TokenIn         common.Address `json:"token_in"`
	TokenOut        common.Address `json:"token_out"`
	AmountIn        *big.Int     `json:"amount_in"`
	AmountOutMin    *big.Int     `json:"amount_out_min"`
}

// ProfitAnalysis 盈利分析结果
type ProfitAnalysis struct {
	TxHash          common.Hash     `json:"tx_hash"`
	TargetContract  common.Address `json:"target_contract"`
	Method          string         `json:"method"`
	Profit          *big.Int       `json:"profit"`        // 预估盈利 (wei)
	GasCost         *big.Int       `json:"gas_cost"`      // Gas成本 (wei)
	NetProfit       *big.Int       `json:"net_profit"`    // 净盈利 (wei)
	SuccessRate     float64        `json:"success_rate"`  // 成功率 (0-1)
	RiskLevel       string         `json:"risk_level"`    // 风险等级
	SimulationTime  int64          `json:"simulation_time"` // 模拟耗时(ms)
	Config          *SniperConfig  `json:"config"`
}

// SniperConfig 狙击手配置（用于类型引用）
type SniperConfig struct {
	MinProfit        *big.Int `json:"min_profit"`
	MaxGasPrice      *big.Int `json:"max_gas_price"`
	MaxGasLimit      uint64   `json:"max_gas_limit"`
}

// ContractInfo 合约信息
type ContractInfo struct {
	Address     common.Address `json:"address"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`        // DEX, Lending, etc.
	ABI         string         `json:"abi"`
	IsVerified  bool           `json:"is_verified"`
}

// MethodSignature 方法签名
type MethodSignature struct {
	Name        string         `json:"name"`
	Selector    []byte         `json:"selector"`
	Parameters  []string       `json:"parameters"`
	IsSwap      bool           `json:"is_swap"`
}

// SwapInfo 交换信息
type SwapInfo struct {
	Dex         string         `json:"dex"`
	Router      common.Address `json:"router"`
	Pair        common.Address `json:"pair"`
	Path        []common.Address `json:"path"`
	Amounts     []*big.Int     `json:"amounts"`
}

// GasEstimation Gas估算结果
type GasEstimation struct {
	GasUsed     uint64         `json:"gas_used"`
	GasPrice    *big.Int       `json:"gas_price"`
	TotalCost   *big.Int       `json:"total_cost"`
	BaseFee     *big.Int       `json:"base_fee"`
	PriorityFee *big.Int       `json:"priority_fee"`
}

// ErrorType 错误类型
type ErrorType struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Level   string `json:"level"` // "info", "warning", "error"
}

// 预定义的合约地址和方法签名
var (
	// 常见DEX路由器地址
	UniswapV2Router = common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")
	UniswapV3Router = common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
	SushiSwapRouter = common.HexToAddress("0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F")

	// 常见交换方法签名
	MethodSwapExactETHForTokens = []byte{0x7f, 0xf3, 0x6a, 0xb5} // swapExactETHForTokens
	MethodSwapExactTokensForETH = []byte{0x18, 0xcb, 0xaf, 0x05} // swapExactTokensForETH
	MethodSwapExactTokensForTokens = []byte{0x38, 0xed, 0x17, 0x39} // swapExactTokensForTokens

	// 支持的DEX列表
	SupportedDEX = map[common.Address]string{
		UniswapV2Router: "Uniswap V2",
		UniswapV3Router: "Uniswap V3", 
		SushiSwapRouter: "SushiSwap",
	}

	// 支持的交换方法
	SupportedSwapMethods = map[string][]byte{
		"swapExactETHForTokens":     MethodSwapExactETHForTokens,
		"swapExactTokensForETH":     MethodSwapExactTokensForETH,
		"swapExactTokensForTokens":  MethodSwapExactTokensForTokens,
	}
)

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