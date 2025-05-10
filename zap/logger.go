package zap

import (
	"encoding/json"
	"fmt"
	"os"

	"net/http"

	"github.com/bt-smart/btlog/loki"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 定义了日志配置
type Config struct {
	// 是否启用控制台日志输出
	EnableConsole bool
	// 是否启用文件日志输出
	EnableFile bool
	// 是否启用Loki输出
	EnableLoki bool
	// 控制台输出的最小日志级别
	ConsoleLevel zapcore.Level
	// 文件输出的最小日志级别
	FileLevel zapcore.Level
	// loki输出的最小日志级别
	LokiLevel zapcore.Level
	// 是否记录调用方信息
	EnableCaller bool
	// 日志文件路径
	FilePath string
	// 日志文件最大大小(MB)
	MaxSize int
	// 保留旧文件的最大个数
	MaxBackups int
	// 保留旧文件的最大天数
	MaxAge int
	// 是否压缩旧文件
	Compress bool
	// Loki配置
	LokiConfig LokiConfig
}

// LokiConfig 定义了Loki相关配置
type LokiConfig struct {
	// Loki服务器地址
	URL string
	// 批量发送大小
	BatchSize int
	// 日志标签
	Labels map[string]string
	// 发送超时时间（秒）
	Timeout int
	// HTTPClient 是用于发送请求的 HTTP 客户端
	// 如果为 nil，将使用 http.DefaultClient
	HTTPClient *http.Client
}

type Logger struct {
	*zap.Logger
	lokiClient *loki.Client
	fileLogger *lumberjack.Logger
}

// NewLogger 创建并返回一个新的日志实例
func NewLogger(cfg *Config) (*Logger, error) {
	var cores []zapcore.Core

	// 使用 zap 预设的 Production 编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	// 控制台输出
	if cfg.EnableConsole {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			cfg.ConsoleLevel,
		)
		cores = append(cores, consoleCore)
	}

	// 文件输出
	var fileLogger *lumberjack.Logger
	if cfg.EnableFile {
		fileLogger = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(fileLogger),
			cfg.FileLevel,
		)
		cores = append(cores, fileCore)
	}

	// 创建并启动 Loki 客户端
	var lokiClient *loki.Client
	if cfg.EnableLoki {
		var err error
		lokiClient, err = loki.NewClient(loki.ClientConfig{
			URL:        cfg.LokiConfig.URL,
			BatchSize:  cfg.LokiConfig.BatchSize,
			Labels:     cfg.LokiConfig.Labels,
			MinLevel:   cfg.LokiLevel,
			HTTPClient: cfg.LokiConfig.HTTPClient,
			// 添加一些合理的默认值
			MinWaitTime: 1,  // 1秒
			MaxWaitTime: 10, // 10秒
		})
		if err != nil {
			return nil, fmt.Errorf("创建 Loki 客户端失败: %v", err)
		}
		lokiClient.Start() // 确保调用 Start()
	}

	core := zapcore.NewTee(cores...)
	// 根据配置决定是否添加调用者信息
	var opts []zap.Option
	if cfg.EnableCaller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	// 创建logger
	logger := zap.New(core, opts...)

	return &Logger{
		Logger:     logger,
		lokiClient: lokiClient,
		fileLogger: fileLogger,
	}, nil
}

// 重写日志方法以支持同时写入Loki
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Debug(formattedMsg)
	}
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Info(formattedMsg)
	}
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Warn(formattedMsg)
	}
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Error(formattedMsg)
	}
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.Logger.DPanic(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Error(formattedMsg) // Loki 没有 DPanic 级别，使用 Error
	}
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Error(formattedMsg) // Loki 没有 Panic 级别，使用 Error
	}
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	if l.lokiClient != nil {
		formattedMsg := formatMessage(msg, fields)
		_ = l.lokiClient.Error(formattedMsg) // Loki 没有 Fatal 级别，使用 Error
	}
	l.Logger.Fatal(msg, fields...) // Fatal 会导致程序退出，所以先发送到 Loki
}

// formatMessage 格式化日志消息，包含字段信息
func formatMessage(msg string, fields []zap.Field) string {
	if len(fields) == 0 {
		return msg
	}

	// 创建一个临时的编码器来格式化字段
	enc := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(enc)
	}

	// 将字段转换为 JSON 字符串
	fieldsJSON, err := json.Marshal(enc.Fields)
	if err != nil {
		return msg
	}

	return fmt.Sprintf("%s %s", msg, string(fieldsJSON))
}

// Close 关闭日志器
func (l *Logger) Close() error {
	// 先同步 zap logger
	err := l.Logger.Sync()

	// 然后关闭 Loki 客户端
	if l.lokiClient != nil {
		l.lokiClient.Stop()
	}

	// 最后关闭文件日志
	if l.fileLogger != nil {
		_ = l.fileLogger.Close()
	}

	return err
}
