// Package pkg 提供了通用的工具组件
package pkg

import (
	"sync"
)

// LogEntry 表示一条日志记录
// 包含时间戳和日志消息两个基本要素
type LogEntry struct {
	// Timestamp 是日志生成时的Unix纳秒时间戳
	// 使用纳秒级时间戳可以保证日志的精确排序
	Timestamp int64

	// Message 存储实际的日志内容
	// 可以是任意字符串消息
	Message string

	// Level 日志级别
	Level LogLevel
}

// Buffer 实现了一个线程安全的日志缓冲区
// 用于批量收集日志条目，当达到指定大小时触发发送
type Buffer struct {
	// entries 存储日志条目
	entries []LogEntry
	// size 是触发发送的目标大小
	size int
	// mu 用于保护并发访问
	mu sync.Mutex
}

// NewBuffer 创建一个新的缓冲区实例
// 参数：
//   - size: 触发发送的目标大小
//
// 返回：
//   - *Buffer: 初始化好的缓冲区实例
func NewBuffer(size int) *Buffer {
	if size <= 0 {
		size = 100 // 设置一个合理的默认值
	}
	return &Buffer{
		entries: make([]LogEntry, 0, size), // 预分配容量以提高性能
		size:    size,
	}
}

// Add 向缓冲区添加一条日志
// 该方法是线程安全的，可以被多个goroutine同时调用
// 参数：
//   - entry: 要添加的日志条目
//
// 返回：
//   - bool: 如果缓冲区达到目标大小返回true，表示应该触发发送操作
func (b *Buffer) Add(entry LogEntry) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 添加日志条目到切片
	b.entries = append(b.entries, entry)

	// 检查是否达到目标大小
	return len(b.entries) >= b.size
}

// Flush 清空并返回缓冲区中的所有日志条目
// 该方法是线程安全的
// 返回：
//   - []LogEntry: 缓冲区中的所有日志条目
func (b *Buffer) Flush() []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 如果没有日志，返回nil
	if len(b.entries) == 0 {
		return nil
	}

	// 获取当前所有日志
	entries := b.entries

	// 创建新的切片，保持预分配的容量
	b.entries = make([]LogEntry, 0, b.size)

	return entries
}
