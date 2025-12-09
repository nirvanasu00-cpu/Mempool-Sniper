package simulator

import (
	"context"
	"log"
	"math/big"
	"sync"
	"time"

	"mempool-sniper/internal/config"
	"mempool-sniper/pkg/types"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Simulator 交易模拟器
type Simulator struct {
	client     *ethclient.Client
	rpcURL     string
	cfg        *config.SniperConfig
	mu         sync.RWMutex
	simulated  int64
	profitable int64
	failed     int64
}

// NewSimulator 创建新的模拟器
func NewSimulator(rpcURL string) *Simulator {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Printf("⚠️ 创建模拟器时连接RPC失败: %v", err)
		// 返回一个无效的模拟器，会在使用时重新连接
		return &Simulator{
			rpcURL: rpcURL,
		}
	}

	return &Simulator{
		client:     client,
		rpcURL:     rpcURL,
		simulated:  0,
		profitable: 0,
		failed:     0,
	}
}

// StartWorkerPool 启动模拟器工作池
func (s *Simulator) StartWorkerPool(ctx context.Context, decodedTxChan <-chan *types.DecodedTransaction, profitChan chan<- *types.ProfitAnalysis, workerCount int) {
	log.Printf("🔮 启动模拟器工作池，工作线程数: %d", workerCount)

	for i := 0; i < workerCount; i++ {
		go s.worker(ctx, decodedTxChan, profitChan, i)
	}
}

// worker 模拟器工作线程
func (s *Simulator) worker(ctx context.Context, decodedTxChan <-chan *types.DecodedTransaction, profitChan chan<- *types.ProfitAnalysis, workerID int) {
	log.Printf("👷 模拟器工作线程 %d 启动", workerID)

	// 确保客户端连接
	if s.client == nil {
		if err := s.reconnect(); err != nil {
			log.Printf("❌ 工作线程 %d 无法连接RPC: %v", workerID, err)
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 模拟器工作线程 %d 停止", workerID)
			return
		case decodedTx := <-decodedTxChan:
			if decodedTx == nil {
				continue
			}

			// 模拟交易执行
			profitAnalysis := s.SimulateTransaction(ctx, decodedTx)
			if profitAnalysis != nil {
				// 将盈利分析结果发送到结果处理器
				select {
				case profitChan <- profitAnalysis:
					log.Printf("💰 工作线程 %d 模拟完成并发送结果: %s -> 盈利 %s ETH",
						workerID, decodedTx.Transaction.Hash.Hex(), profitAnalysis.NetProfit.String())
				case <-ctx.Done():
					return
				default:
					log.Printf("⚠️ 工作线程 %d 盈利通道已满，丢弃结果: %s", workerID, decodedTx.Transaction.Hash.Hex())
				}
			}
		}
	}
}

// SimulateTransaction 模拟交易执行
func (s *Simulator) SimulateTransaction(ctx context.Context, decodedTx *types.DecodedTransaction) *types.ProfitAnalysis {
	startTime := time.Now()

	s.mu.Lock()
	s.simulated++
	s.mu.Unlock()

	// 确保客户端连接
	if s.client == nil {
		if err := s.reconnect(); err != nil {
			s.mu.Lock()
			s.failed++
			s.mu.Unlock()
			return nil
		}
	}

	// 简化版模拟逻辑
	// 实际项目中需要实现完整的EVM模拟
	profitAnalysis := &types.ProfitAnalysis{
		TxHash:         decodedTx.Transaction.Hash,
		TargetContract: decodedTx.TargetContract,
		Method:         decodedTx.Method,
		SimulationTime: time.Since(startTime).Milliseconds(),
	}

	// 估算Gas成本
	gasCost := s.estimateGasCost(decodedTx)
	profitAnalysis.GasCost = gasCost

	// 简化盈利计算
	// 实际项目中需要根据具体交易进行精确计算
	profit := s.calculateProfit(decodedTx, gasCost)
	profitAnalysis.Profit = profit
	profitAnalysis.NetProfit = new(big.Int).Sub(profit, gasCost)

	// 计算成功率（简化）
	profitAnalysis.SuccessRate = s.calculateSuccessRate(decodedTx)

	// 评估风险等级
	profitAnalysis.RiskLevel = s.assessRiskLevel(decodedTx, profitAnalysis.SuccessRate)

	s.mu.Lock()
	if profitAnalysis.NetProfit.Cmp(big.NewInt(0)) > 0 {
		s.profitable++
	}
	s.mu.Unlock()

	return profitAnalysis
}

// estimateGasCost 估算Gas成本
func (s *Simulator) estimateGasCost(decodedTx *types.DecodedTransaction) *big.Int {
	// 简化Gas估算
	// 实际项目中需要根据交易复杂度进行精确估算
	baseGas := uint64(21000)       // 基础Gas
	additionalGas := uint64(50000) // 交换操作额外Gas

	totalGas := baseGas + additionalGas

	// 使用交易中的Gas价格或当前网络平均Gas价格
	gasPrice := decodedTx.Transaction.GasPrice
	if gasPrice == nil || gasPrice.Cmp(big.NewInt(0)) == 0 {
		// 如果没有Gas价格，使用默认值
		gasPrice = big.NewInt(30000000000) // 30 Gwei
	}

	gasCost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(totalGas))
	return gasCost
}

// calculateProfit 计算盈利
func (s *Simulator) calculateProfit(decodedTx *types.DecodedTransaction, gasCost *big.Int) *big.Int {
	// 简化盈利计算
	// 实际项目中需要根据具体交易对和价格进行精确计算

	// 基于交易金额的百分比估算盈利
	baseProfit := new(big.Int).Div(decodedTx.Transaction.Value, big.NewInt(100)) // 1%

	// 减去Gas成本
	netProfit := new(big.Int).Sub(baseProfit, gasCost)

	// 确保盈利不为负
	if netProfit.Cmp(big.NewInt(0)) < 0 {
		return big.NewInt(0)
	}

	return netProfit
}

// calculateSuccessRate 计算成功率
func (s *Simulator) calculateSuccessRate(decodedTx *types.DecodedTransaction) float64 {
	// 简化成功率计算
	// 实际项目中需要根据历史数据和市场条件计算

	baseRate := 0.8 // 基础成功率80%

	// 根据交易金额调整成功率
	// 大额交易成功率较低
	value := decodedTx.Transaction.Value
	if value.Cmp(big.NewInt(1000000000000000000)) > 0 { // 大于1 ETH
		baseRate *= 0.7
	}

	// 根据Gas价格调整成功率
	gasPrice := decodedTx.Transaction.GasPrice
	if gasPrice != nil && gasPrice.Cmp(big.NewInt(100000000000)) > 0 { // 大于100 Gwei
		baseRate *= 0.9
	}

	return baseRate
}

// assessRiskLevel 评估风险等级
func (s *Simulator) assessRiskLevel(decodedTx *types.DecodedTransaction, successRate float64) string {
	if successRate >= 0.9 {
		return "low"
	} else if successRate >= 0.7 {
		return "medium"
	} else {
		return "high"
	}
}

// reconnect 重新连接RPC
func (s *Simulator) reconnect() error {
	client, err := ethclient.Dial(s.rpcURL)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.client = client
	s.mu.Unlock()

	log.Println("✅ 模拟器RPC连接成功")
	return nil
}

// GetStats 获取统计信息
func (s *Simulator) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	successRate := float64(0)
	if s.simulated > 0 {
		successRate = float64(s.simulated-s.failed) / float64(s.simulated) * 100
	}

	profitabilityRate := float64(0)
	if s.simulated > 0 {
		profitabilityRate = float64(s.profitable) / float64(s.simulated) * 100
	}

	return map[string]interface{}{
		"simulated":          s.simulated,
		"profitable":         s.profitable,
		"failed":             s.failed,
		"success_rate":       successRate,
		"profitability_rate": profitabilityRate,
		"rpc_url":            s.rpcURL,
	}
}

// IsConnected 检查是否已连接
func (s *Simulator) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil
}

// SetConfig 设置配置
func (s *Simulator) SetConfig(cfg *config.SniperConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

// AdvancedSimulation 高级模拟（预留接口）
func (s *Simulator) AdvancedSimulation(ctx context.Context, decodedTx *types.DecodedTransaction) *types.ProfitAnalysis {
	// 这里可以实现更复杂的模拟逻辑
	// 包括本地EVM执行、状态fork等高级功能

	// 目前先调用基础模拟
	return s.SimulateTransaction(ctx, decodedTx)
}
