# btlog

一个基于 [zap](https://github.com/uber-go/zap) 实现的日志工具

## 功能特性

- 支持同时输出到控制台和文件
- 支持日志级别控制
- 支持日志文件自动切割
- 支持调用者信息记录
- 控制台采用开发友好格式，文件输出采用 JSON 格式
- 支持结构化字段记录

## 依赖库

- [go.uber.org/zap](https://github.com/uber-go/zap) - 高性能日志库
- [gopkg.in/natefinch/lumberjack.v2](https://github.com/natefinch/lumberjack) - 日志切割库

## 安装

```bash
go get github.com/bt-smart/btlog
```

## 使用

请参考 [example](example) 目录下的示例代码。
