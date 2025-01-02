package btlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	// 是否启用控制台日志输出
	EnableConsole bool
	// 控制台输出的最小日志级别
	ConsoleLevel zapcore.Level
	// 文件输出的最小日志级别
	FileLevel zapcore.Level
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
}

type Logger struct {
	zapLogger *zap.Logger
	hook      *lumberjack.Logger
}

// NewLogger 创建并返回一个新的日志实例
func NewLogger(cfg *Config) (*Logger, error) {
	// 创建lumberjack的日志切割配置
	hook := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 创建两个core，分别用于控制台和文件输出
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// 创建core，根据配置决定是否包含控制台输出
	var cores []zapcore.Core
	if cfg.EnableConsole {
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), cfg.ConsoleLevel))
	}
	cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(hook), cfg.FileLevel))

	core := zapcore.NewTee(cores...)

	// 根据配置决定是否添加调用者信息
	var opts []zap.Option
	if cfg.EnableCaller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	// 创建logger
	zapLogger := zap.New(core, opts...)
	return &Logger{
		zapLogger: zapLogger,
		hook:      hook,
	}, nil
}

// Sync 同步日志缓冲区到磁盘
func (l *Logger) Sync() error {
	if l.zapLogger != nil {
		return l.zapLogger.Sync()
	}
	return nil
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if err := l.Sync(); err != nil {
		return err
	}
	if l.hook != nil {
		return l.hook.Close()
	}
	return nil
}

// Info 打印info级别日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
}

// Warn 打印warn级别日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
}

// Error 打印error级别日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, fields...)
}

// Panic 打印panic级别日志
func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zapLogger.Panic(msg, fields...)
}

// GetZapLogger 返回底层的 zap.Logger 实例
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.zapLogger
}
