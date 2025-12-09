package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mempool-sniper/internal/config"
	"mempool-sniper/internal/decoder"
	"mempool-sniper/internal/listener"
	"mempool-sniper/internal/simulator"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 创建监听器
	listener := listener.NewListener(cfg)
	decoder := decoder.NewDecoder(cfg)
	simulator := simulator.NewSimulator(cfg)

	// 启动组件
	if err := listener.Start(ctx); err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}

	decoder.StartWorkerPool()
	simulator.StartWorkerPool()

	// 设置交易处理管道
	transactionCh := listener.GetTransactionChannel()
	decodedCh := decoder.GetDecodedChannel()

	// 启动处理协程
	go func() {
		for {
			select {
			case tx := <-transactionCh:
				// 发送到解码器
				decoder.SubmitTransaction(tx)
			case decodedTx := <-decodedCh:
				// 发送到模拟器
				simulator.SubmitTransaction(decodedTx)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 启动统计报告
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Printf("Stats - Transactions: %d, Decoded: %d, Simulated: %d, Profitable: %d\n",
					listener.GetStats().TotalTransactions,
					decoder.GetStats().TotalDecoded,
					simulator.GetStats().TotalSimulated,
					simulator.GetStats().ProfitableTransactions)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Println("Mempool Sniper started successfully")

	// 等待退出信号
	select {
	case <-sigCh:
		log.Println("Received shutdown signal")
	case <-ctx.Done():
	}

	// 优雅关闭
	listener.Stop()
	decoder.Stop()
	simulator.Stop()
	
	log.Println("Mempool Sniper stopped")
}