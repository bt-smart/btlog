package pkg

// LogLevel 定义日志级别
type LogLevel int

const (
	// 定义日志级别常量
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// LevelToString 将日志级别转换为字符串
func LevelToString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}
