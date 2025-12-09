package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"math/big"

	"mempool-sniper/internal/config"
	"mempool-sniper/internal/decoder"
	"mempool-sniper/internal/listener"
	"mempool-sniper/internal/simulator"
	"mempool-sniper/pkg/types"
)

// SniperConfig 狙击手配置（用于类型引用）
type SniperConfig struct {
	MinProfit   *big.Int `json:"min_profit"`
	MaxGasPrice *big.Int `json:"max_gas_price"`
	MaxGasLimit uint64   `json:"max_gas_limit"`
}

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	setupSignalHandler(cancel)

	// 创建监听器
	listener, err := listener.NewListener(cfg.Ethereum.WSSURL)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	// 创建解码器
	decoder := decoder.NewDecoder()

	// 创建模拟器
	simulator := simulator.NewSimulator(cfg.Ethereum.RPCURL)

	// 创建交易通道和盈利分析通道
	txChan := make(chan *types.Transaction, 100)
	profitChan := make(chan *types.ProfitAnalysis, 100)

	// 启动监听器
	go listener.Start(ctx, txChan)

	// 启动解码器工作池
	go decoder.StartWorkerPool(ctx, txChan, 5)

	// 启动模拟器工作池
	go simulator.StartWorkerPool(ctx, profitChan, 3)

	// 启动结果处理器
	go processResults(ctx, profitChan)

	log.Println("🚀 Mempool Sniper 启动成功")
	log.Printf("📡 监听节点: %s", cfg.Ethereum.WSSURL)
	log.Printf("🔍 模拟节点: %s", cfg.Ethereum.RPCURL)
	log.Println("⏳ 等待交易...")

	// 等待程序退出
	<-ctx.Done()
	log.Println("🛑 Mempool Sniper 正在关闭...")

	// 等待所有goroutine完成
	time.Sleep(2 * time.Second)
	log.Println("✅ Mempool Sniper 已安全关闭")
}

// setupSignalHandler 设置信号处理器
func setupSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("📡 收到信号: %v", sig)
		cancel()
	}()
}

// processResults 处理盈利分析结果
func processResults(ctx context.Context, profitChan chan *types.ProfitAnalysis) {
	for {
		select {
		case <-ctx.Done():
			return
		case analysis := <-profitChan:
			if analysis != nil && analysis.Profit.Cmp(analysis.Config.MinProfit) >= 0 {
				log.Printf("💰 发现盈利机会!")
				log.Printf("  交易哈希: %s", analysis.TxHash.Hex())
				log.Printf("  预估盈利: %s ETH", analysis.Profit.String())
				log.Printf("  目标合约: %s", analysis.TargetContract.Hex())
				log.Printf("  方法: %s", analysis.Method)

				// 这里可以添加自动交易逻辑
				// 或者发送通知到外部系统
			}
		}
	}
}
