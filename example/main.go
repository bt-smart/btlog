package main

import (
	"fmt"
	"os"

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

	if err := btlog.InitLogger(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := btlog.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "关闭日志器失败: %v\n", err)
		}
	}()

	btlog.Info("应用启动成功",
		zap.String("env", "development"),
		zap.Int("pid", os.Getpid()),
	)

	btlog.Warn("系统资源使用率较高",
		zap.Int("cpu_usage", 85),
		zap.Int("memory_usage", 90),
	)

	btlog.Error("数据库连接失败",
		zap.String("db_host", "localhost"),
		zap.Int("port", 5432),
	)
}
