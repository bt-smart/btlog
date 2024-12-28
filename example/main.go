package main

import (
	"github.com/bt-smart/btlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	cfg := &btlog.Config{
		EnableConsole: true,              // 是否启用控制台日志输出
		ConsoleLevel:  zapcore.InfoLevel, // 控制台输出的最小日志级别
		FileLevel:     zapcore.WarnLevel, // 文件输出的最小日志级别
		EnableCaller:  true,              // 是否记录调用方信息
		FilePath:      "./logs/app.log",  // 日志文件路径
		MaxSize:       100,               // 日志文件最大大小(MB)
		MaxBackups:    30,                // 保留旧文件的最大个数
		MaxAge:        7,                 // 保留旧文件的最大天数
		Compress:      true,              // 是否压缩旧文件
	}

	err := btlog.InitLogger(cfg)
	if err != nil {
		panic(err)
	}

	btlog.Info("这是一条信息日志")
	btlog.Warn("这是一条警告日志", zap.String("key", "value"))
	btlog.Error("这是一条错误日志", zap.Int("code", 500))
}
