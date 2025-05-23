package main

import (
	"net/http"
	"time"

	btzap "github.com/bt-smart/btlog/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 创建共享的 HTTP 客户端
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	cfg := &btzap.Config{
		EnableConsole: true,              // 是否启用控制台日志输出
		EnableFile:    false,             // 是否启用文件日志输出
		EnableLoki:    true,              // 是否启用Loki日志输出
		ConsoleLevel:  zapcore.InfoLevel, // 控制台输出的最小日志级别
		FileLevel:     zapcore.InfoLevel, // 文件输出的最小日志级别
		LokiLevel:     zapcore.InfoLevel, // loki输出的最小日志级别
		EnableCaller:  true,              // 是否记录调用方信息
		FilePath:      "./logs/app.log",  // 日志文件路径
		MaxSize:       100,               // 日志文件最大大小(MB)
		MaxBackups:    3,                 // 保留旧文件的最大个数
		MaxAge:        28,                // 保留旧文件的最大天数
		Compress:      true,              // 是否压缩旧文件
		LokiConfig: btzap.LokiConfig{ // Loki配置
			URL:        "http://192.168.98.214:3100",
			BatchSize:  100,
			Labels:     map[string]string{"service_name": "btlog-demo-dev"},
			HTTPClient: httpClient,
		},
	}

	logger, err := btzap.NewLogger(cfg)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Info("这是一条信息日志")
	logger.Error("这是一条错误日志", zap.String("error", "发生错误"))
}
