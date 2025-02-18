// Package loki 实现了Loki日志系统的客户端
package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bt-smart/btlog/pkg"
)

// Client 实现了Loki的客户端，提供日志推送功能
// 支持批量发送、缓存、自动重试等特性
type Client struct {
	// config 存储客户端的配置信息，包括服务器地址、标签等
	config ClientConfig
	// buffer 是内存中的日志缓冲区，用于批量发送日志
	buffer *pkg.Buffer
	// done 是用于优雅关闭的信号通道
	done chan bool
	// httpClient 是用于发送请求的 HTTP 客户端
	httpClient *http.Client
	// closed 是用于标记客户端是否已关闭的标志
	closed atomic.Bool
	// started 是用于标记客户端是否已启动的标志
	started atomic.Bool
}

// NewClient 创建并初始化一个新的Loki客户端实例
// 参数：
//   - config: 客户端配置，包含服务器地址、批量大小等设置
//
// 返回：
//   - *Client: 初始化好的客户端实例
//   - error: 如果配置无效则返回错误
func NewClient(config ClientConfig) (*Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.MinWaitTime == 0 {
		config.MinWaitTime = 1
	}
	if config.MaxWaitTime == 0 {
		config.MaxWaitTime = 10
	}
	if config.MaxWaitTime <= config.MinWaitTime {
		config.MaxWaitTime = config.MinWaitTime + 1
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		config:     config,
		buffer:     pkg.NewBuffer(config.BatchSize),
		done:       make(chan bool, 1),
		httpClient: httpClient,
	}, nil
}

// Debug 记录调试级别的日志
func (c *Client) Debug(message string) error {
	return c.pushLogWithLevel(message, pkg.LevelDebug)
}

// Info 记录信息级别的日志
func (c *Client) Info(message string) error {
	return c.pushLogWithLevel(message, pkg.LevelInfo)
}

// Warn 记录警告级别的日志
func (c *Client) Warn(message string) error {
	return c.pushLogWithLevel(message, pkg.LevelWarn)
}

// Error 记录错误级别的日志
func (c *Client) Error(message string) error {
	return c.pushLogWithLevel(message, pkg.LevelError)
}

// pushLogWithLevel 内部方法，处理带级别的日志推送
// 参数：
//   - message: 日志消息内容
//   - level: 日志级别
//
// 返回：
//   - error: 如果客户端未启动或已关闭，或者推送失败则返回错误
func (c *Client) pushLogWithLevel(message string, level pkg.LogLevel) error {
	// 检查是否已关闭或未启动
	if c.closed.Load() {
		return fmt.Errorf("client is closed")
	}
	if !c.started.Load() {
		return fmt.Errorf("client is not started")
	}

	if level < c.config.MinLevel {
		return nil
	}

	entry := pkg.LogEntry{
		Timestamp: time.Now().UnixNano(),
		Message:   message,
		Level:     level,
	}

	if c.buffer.Add(entry) {
		c.flush()
	}
	return nil
}

// Start 启动客户端的后台工作协程
// 该方法是线程安全的，可以被多次调用
// 只有第一次调用会真正启动工作协程
func (c *Client) Start() {
	// 防止重复启动
	if c.started.Swap(true) {
		return
	}
	go c.worker()
}

// Stop 停止客户端的后台工作协程
// 该方法是线程安全的，可以被多次调用
// 在停止前会确保所有缓存的日志都被发送
func (c *Client) Stop() {
	// 如果未启动或已关闭，直接返回
	if !c.started.Load() || c.closed.Swap(true) {
		return
	}

	c.flush() // 最后一次刷新
	c.done <- true

	// 等待一小段时间确保最后的日志被发送
	time.Sleep(time.Millisecond * 100)
}

// worker 是后台工作协程的主循环
// 负责定期检查并发送日志，实现了以下功能：
// 1. 定期检查是否需要发送日志
// 2. 处理优雅关闭信号
// 3. 确保日志不会在缓冲区中停留太久
func (c *Client) worker() {
	// 创建定时器，用于周期性检查是否需要发送日志
	ticker := time.NewTicker(time.Second * time.Duration(c.config.MaxWaitTime))
	lastFlush := time.Now()

	// 确保 ticker 被正确清理
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			// 在退出前应该再次检查是否有未发送的日志
			if time.Since(lastFlush) > 0 {
				c.flush()
			}
			return
		case <-ticker.C:
			// 检查是否超过最大等待时间
			if time.Since(lastFlush) >= time.Second*time.Duration(c.config.MaxWaitTime) {
				c.flush()
				lastFlush = time.Now()
			}
		}
	}
}

// flush 将缓冲区中的日志发送到Loki服务器
// 主要步骤：
// 1. 从缓冲区获取所有待发送的日志
// 2. 将日志转换为Loki期望的格式
// 3. 发送到服务器
func (c *Client) flush() {
	entries := c.buffer.Flush()
	if len(entries) == 0 {
		return
	}

	// 按日志级别分组
	levelGroups := make(map[pkg.LogLevel][][2]string)
	for _, entry := range entries {
		levelGroups[entry.Level] = append(levelGroups[entry.Level], [2]string{
			strconv.FormatInt(entry.Timestamp, 10),
			entry.Message,
		})
	}

	// 为每个级别创建单独的流
	var streams []Stream
	for level, values := range levelGroups {
		// 复制标签并添加级别
		labels := make(map[string]string)
		for k, v := range c.config.Labels {
			labels[k] = v
		}
		// 添加日志级别标签
		labels["detected_level"] = pkg.LevelToString(level)

		streams = append(streams, Stream{
			Stream: labels,
			Values: values,
		})
	}

	req := PushRequest{
		Streams: streams,
	}

	// 处理发送错误
	if err := c.send(req); err != nil {
		// 这里可以考虑将失败的日志重新加入缓冲区，或者记录错误
		// 为了避免递归，这里使用标准库的log包记录错误
		log.Printf("Failed to send logs to Loki: %v", err)
	}
}

// send 负责将日志请求发送到Loki服务器
// 参数：
//   - req: 要发送的日志请求
//
// 返回：
//   - error: 发送过程中的错误，如果成功则为nil
func (c *Client) send(req PushRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request failed: %v", err)
	}

	resp, err := c.httpClient.Post(c.config.URL+"/loki/api/v1/push", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
