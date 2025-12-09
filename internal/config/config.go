package config

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置结构体
type Config struct {
	Ethereum EthereumConfig `json:"ethereum"`
	Sniper   SniperConfig   `json:"sniper"`
	Logging  LoggingConfig  `json:"logging"`
}

// EthereumConfig Ethereum节点配置
type EthereumConfig struct {
	WSSURL  string `json:"wss_url"`
	RPCURL  string `json:"rpc_url"`
	ChainID int64  `json:"chain_id"`
}

// SniperConfig 狙击手配置
type SniperConfig struct {
	MinProfit         *big.Int `json:"min_profit"`         // 最小盈利阈值 (wei)
	MaxGasPrice       *big.Int `json:"max_gas_price"`      // 最大Gas价格
	MaxGasLimit       uint64   `json:"max_gas_limit"`      // 最大Gas限制
	WorkerPoolSize    int      `json:"worker_pool_size"`   // 工作池大小
	SimulationTimeout int      `json:"simulation_timeout"` // 模拟超时(秒)
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level    string `json:"level"`     // 日志级别
	FilePath string `json:"file_path"` // 日志文件路径
}

// Load 加载配置
func Load() (*Config, error) {
	// 加载.env文件（如果存在）
	_ = godotenv.Load()

	cfg := &Config{
		Ethereum: EthereumConfig{
			WSSURL:  getEnv("ETH_WSS_URL", "wss://mainnet.infura.io/ws/v3/YOUR_INFURA_PROJECT_ID"),
			RPCURL:  getEnv("ETH_RPC_URL", "https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID"),
			ChainID: getEnvInt64("ETH_CHAIN_ID", 1),
		},
		Sniper: SniperConfig{
			MinProfit:         getEnvBigInt("MIN_PROFIT", "1000000000000000"), // 0.001 ETH
			MaxGasPrice:       getEnvBigInt("MAX_GAS_PRICE", "50000000000"),   // 50 Gwei
			MaxGasLimit:       getEnvUint64("MAX_GAS_LIMIT", 300000),
			WorkerPoolSize:    getEnvInt("WORKER_POOL_SIZE", 5),
			SimulationTimeout: getEnvInt("SIMULATION_TIMEOUT", 10),
		},
		Logging: LoggingConfig{
			Level:    getEnv("LOG_LEVEL", "info"),
			FilePath: getEnv("LOG_FILE", "mempool-sniper.log"),
		},
	}

	// 验证配置
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate 验证配置
func (c *Config) validate() error {
	if c.Ethereum.WSSURL == "" || c.Ethereum.WSSURL == "wss://mainnet.infura.io/ws/v3/YOUR_INFURA_PROJECT_ID" {
		return fmt.Errorf("ETH_WSS_URL 必须配置为有效的WebSocket URL")
	}

	if c.Ethereum.RPCURL == "" || c.Ethereum.RPCURL == "https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID" {
		return fmt.Errorf("ETH_RPC_URL 必须配置为有效的RPC URL")
	}

	if c.Sniper.MinProfit.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("MIN_PROFIT 必须大于0")
	}

	if c.Sniper.MaxGasPrice.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("MAX_GAS_PRICE 必须大于0")
	}

	if c.Sniper.MaxGasLimit == 0 {
		return fmt.Errorf("MAX_GAS_LIMIT 必须大于0")
	}

	if c.Sniper.WorkerPoolSize <= 0 {
		return fmt.Errorf("WORKER_POOL_SIZE 必须大于0")
	}

	return nil
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uintValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uintValue
		}
	}
	return defaultValue
}

func getEnvBigInt(key string, defaultValue string) *big.Int {
	if value := os.Getenv(key); value != "" {
		if bigIntValue, ok := new(big.Int).SetString(value, 10); ok {
			return bigIntValue
		}
	}

	bigIntValue, _ := new(big.Int).SetString(defaultValue, 10)
	return bigIntValue
}
