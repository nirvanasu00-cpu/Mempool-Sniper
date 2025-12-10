package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"mempool-sniper/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Listener 交易监听器
type Listener struct {
	client    *ethclient.Client
	rpcClient *rpc.Client
	wssURL    string
	isRunning bool
	mu        sync.RWMutex
	txCount   int64
	startTime time.Time
}

// NewListener 创建新的监听器
func NewListener(wssURL string) (*Listener, error) {
	client, err := ethclient.Dial(wssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %v", err)
	}

	rpcClient, err := rpc.Dial(wssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %v", err)
	}

	return &Listener{
		client:    client,
		rpcClient: rpcClient,
		wssURL:    wssURL,
		isRunning: false,
		txCount:   0,
		startTime: time.Now(),
	}, nil
}

// Start 启动监听器
func (l *Listener) Start(ctx context.Context, txChan chan<- *types.Transaction) error {
	l.mu.Lock()
	if l.isRunning {
		l.mu.Unlock()
		return fmt.Errorf("listener is already running")
	}
	l.isRunning = true
	l.startTime = time.Now()
	l.mu.Unlock()

	log.Printf("📡 开始监听内存池交易...")

	// 创建新区块通道
	// 注：这是个经验值：100 个区块大约涵盖了 20 分钟的数据量，这给下游处理留足了‘喘息时间’，既保证了不阻塞网络 IO，又不会占用过多内存。
	headChan := make(chan *ethtypes.Header, 100)

	// 订阅新区块
	headSub, err := l.client.SubscribeNewHead(ctx, headChan)
	if err != nil {
		l.mu.Lock()
		l.isRunning = false
		l.mu.Unlock()
		return fmt.Errorf("failed to subscribe to new heads: %v", err)
	}

	// 启动区块处理goroutine
	go l.processHeads(ctx, headChan, txChan)

	// 启动pending交易监听goroutine
	go l.subscribePendingTransactions(ctx, txChan)

	// 处理订阅事件
	go func() {
		defer headSub.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				log.Println("🛑 监听器收到停止信号")
				l.mu.Lock()
				l.isRunning = false
				l.mu.Unlock()
				return
			case err := <-headSub.Err():
				log.Printf("⚠️ 新区块订阅错误: %v", err)
				// 尝试重新连接
				go l.reconnect(ctx, txChan)
				return
			}
		}
	}()

	return nil
}

// subscribePendingTransactions 订阅pending交易（改进版：支持自动重连）
func (l *Listener) subscribePendingTransactions(ctx context.Context, txChan chan<- *types.Transaction) {
	// 指数退避配置
	backoff := time.Second
	maxBackoff := 30 * time.Second
	retryCount := 0

	for {
		retryCount++

		// 检查是否被主动停止
		select {
		case <-ctx.Done():
			log.Println("🛑 Pending交易订阅收到停止信号")
			return
		default:
		}

		// 使用rpc客户端订阅pending交易
		pendingTxChan := make(chan string, 1000)

		sub, err := l.rpcClient.EthSubscribe(ctx, pendingTxChan, "newPendingTransactions")
		if err != nil {
			log.Printf("❌ 无法订阅pending交易 (尝试 %d), %v后重试: %v",
				retryCount, backoff, err)

			// 指数退避等待
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			// 更新退避时间
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		log.Printf("✅ 第%d次订阅pending交易成功", retryCount)

		// 连接成功后重置退避时间
		backoff = time.Second
		retryCount = 0

		// 处理订阅事件
		func() {
			defer sub.Unsubscribe()

			for {
				select {
				case <-ctx.Done():
					log.Println("🛑 Pending交易订阅内部处理收到停止信号")
					return
				case err := <-sub.Err():
					log.Printf("⚠️ Pending交易订阅错误，触发重连: %v", err)
					return
				case txHashStr := <-pendingTxChan:
					if txHashStr == "" {
						continue
					}

					// 检查是否被主动停止
					select {
					case <-ctx.Done():
						log.Println("🛑 Pending交易订阅处理交易时收到停止信号")
						return
					default:
					}

					// 将字符串转换为Hash
					txHash := common.HexToHash(txHashStr)

					// 打印pending交易日志
					log.Printf("[PENDING] 收到交易: %s", txHash.Hex())

					// 异步处理交易
					go l.fetchAndProcessTransaction(ctx, txHash, txChan)
				}
			}
		}()

		// 订阅断开后，继续外层循环进行重连
		log.Println("🔄 Pending交易订阅断开，准备重连...")
	}
}

// processHeads 处理新区块
func (l *Listener) processHeads(ctx context.Context, headChan <-chan *ethtypes.Header, txChan chan<- *types.Transaction) {
	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 区块处理goroutine收到停止信号")
			return
		case header := <-headChan:
			if header == nil {
				continue
			}

			// 当新区块到达时，获取当前pending transactions
			go l.fetchPendingTransactions(ctx, header.Number, txChan)
		}
	}
}

// fetchPendingTransactions 获取pending transactions
func (l *Listener) fetchPendingTransactions(ctx context.Context, blockNumber *big.Int, txChan chan<- *types.Transaction) {
	// 检查是否被主动停止
	select {
	case <-ctx.Done():
		log.Println("🛑 fetchPendingTransactions收到停止信号")
		return
	default:
	}
	
	log.Printf("📦 新区块到达: %s", blockNumber.String())
	// 现在有了SubscribePendingTransactions，此函数主要用于区块到达时的处理
}

// fetchAndProcessTransaction 获取并处理交易
func (l *Listener) fetchAndProcessTransaction(ctx context.Context, txHash common.Hash, txChan chan<- *types.Transaction) {
	// 检查是否被主动停止
	select {
	case <-ctx.Done():
		log.Println("🛑 fetchAndProcessTransaction收到停止信号")
		return
	default:
	}

	// 重试机制
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			log.Println("🛑 fetchAndProcessTransaction重试过程中收到停止信号")
			return
		default:
			tx, isPending, err := l.client.TransactionByHash(ctx, txHash)
			if err != nil {
				// 交易可能已被丢弃，等待后重试
				select {
				case <-ctx.Done():
					log.Println("🛑 fetchAndProcessTransaction等待重试时收到停止信号")
					return
				case <-time.After(time.Duration(i+1) * 100 * time.Millisecond):
				}
				continue
			}

			if !isPending {
				// 交易已打包，跳过
				return
			}

			// 创建交易对象
			transaction := &types.Transaction{
				Hash:      tx.Hash(),
				RawTx:     tx,
				To:        tx.To(),
				Value:     tx.Value(),
				GasPrice:  tx.GasPrice(),
				GasLimit:  tx.Gas(),
				Data:      tx.Data(),
				Nonce:     tx.Nonce(),
				ChainID:   tx.ChainId(),
				Timestamp: time.Now().Unix(),
			}

			// 尝试获取发送者地址
			signer := ethtypes.NewLondonSigner(tx.ChainId())
			if from, err := signer.Sender(tx); err == nil {
				transaction.From = from
			}

			// 发送到处理通道（非阻塞发送，避免缓冲区满时阻塞）
				select {
				case txChan <- transaction:
					// 更新交易计数
					l.mu.Lock()
					l.txCount++
					l.mu.Unlock()

					// 打印处理成功的日志
					toAddress := "合约创建"
					if transaction.To != nil {
						toAddress = transaction.To.Hex()[:10] + "..."
					}
					log.Printf("[PENDING] 处理成功: %s (From: %s, To: %s, Value: %s ETH)",
						txHash.Hex()[:10]+"...",
						transaction.From.Hex()[:10]+"...",
						toAddress,
						transaction.Value.String())

					// 统计信息（每100笔交易打印一次）
					if l.txCount%100 == 0 {
						l.logStats()
					}
					return
				case <-ctx.Done():
					log.Println("🛑 fetchAndProcessTransaction发送交易时收到停止信号")
					return
				default:
					log.Printf("⚠️ 交易通道已满，丢弃交易: %s", txHash.Hex()[:10]+"...")
					return
				}
		}
	}

	log.Printf("❌ 无法获取交易: %s (重试3次失败)", txHash.Hex()[:10]+"...")
}

// reconnect 重新连接（改进版：无限重连 + 指数退避）
func (l *Listener) reconnect(ctx context.Context, txChan chan<- *types.Transaction) {
	log.Println("🔄 检测到连接断开，启动自动重连...")

	// 指数退避配置
	backoff := time.Second
	maxBackoff := 30 * time.Second
	retryCount := 0

	for {
		retryCount++

		// 检查是否被主动停止
		select {
		case <-ctx.Done():
			log.Println("🛑 重连过程被主动停止")
			return
		default:
		}

		// 尝试重新连接
		newListener, err := NewListener(l.wssURL)
		if err != nil {
			log.Printf("❌ 重连失败 (尝试 %d), %v后重试: %v",
				retryCount, backoff, err)

			// 指数退避等待
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			// 更新退避时间
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// 替换旧的监听器（简单替换，避免热切换的复杂度）
		l.mu.Lock()
		if l.client != nil {
			l.client.Close()
		}
		l.client = newListener.client
		l.rpcClient = newListener.rpcClient
		l.isRunning = true
		l.mu.Unlock()

		log.Printf("✅ 第%d次重连成功，重置退避时间", retryCount)

		// 连接成功后重置退避时间
		backoff = time.Second

		// 重新启动订阅（这里需要确保不会重复启动）
		// 由于Start函数有运行状态检查，这里直接返回，由外层的错误处理重新调用Start
		return
	}
}

// logStats 记录统计信息
func (l *Listener) logStats() {
	l.mu.RLock()
	defer l.mu.RUnlock()

	duration := time.Since(l.startTime)
	tps := float64(l.txCount) / duration.Seconds()

	log.Printf("📊 统计信息 - 总交易数: %d, 运行时间: %v, TPS: %.2f",
		l.txCount, duration.Round(time.Second), tps)
}

// GetStats 获取统计信息
func (l *Listener) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	duration := time.Since(l.startTime)
	tps := float64(l.txCount) / duration.Seconds()

	return map[string]interface{}{
		"is_running": l.isRunning,
		"tx_count":   l.txCount,
		"start_time": l.startTime,
		"duration":   duration,
		"tps":        tps,
		"wss_url":    l.wssURL,
	}
}

// Stop 停止监听器
func (l *Listener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isRunning {
		l.isRunning = false
		if l.client != nil {
			l.client.Close()
		}
		if l.rpcClient != nil {
			l.rpcClient.Close()
		}
		log.Println("🛑 监听器已停止")
	}
}

// IsRunning 检查监听器是否在运行
func (l *Listener) IsRunning() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.isRunning
}