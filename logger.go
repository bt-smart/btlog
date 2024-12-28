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

var (
	logger *zap.Logger
	hook   *lumberjack.Logger
)

// sync 同步日志缓冲区到磁盘
// 如果logger已初始化,则调用其Sync方法将缓冲区数据刷新到磁盘
// 如果logger未初始化,则直接返回nil
func sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// Close 关闭日志文件
// 首先调用sync()方法将缓冲区数据刷新到磁盘
// 如果sync()返回错误则直接返回该错误
// 如果hook不为nil,则调用其Close()方法关闭日志文件
// 返回关闭文件时可能产生的错误
func Close() error {
	if err := sync(); err != nil {
		return err
	}
	if hook != nil {
		return hook.Close()
	}
	return nil
}

// InitLogger 初始化日志配置
func InitLogger(cfg *Config) error {
	// 创建lumberjack的日志切割配置
	hook = &lumberjack.Logger{
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
	logger = zap.New(core, opts...)
	return nil
}

// Info 打印info级别日志
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Warn 打印warn级别日志
func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// Error 打印error级别日志
func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// Panic 打印panic级别日志
func Panic(msg string, fields ...zap.Field) {
	logger.Panic(msg, fields...)
}
