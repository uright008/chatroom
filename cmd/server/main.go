package main

import (
	"log"
	"os"
	"syscall"
	"os/signal"

	"go-chatroom/internal/config"
	"go-chatroom/internal/server"
)

func main() {
	// 初始化配置
	cfg, err := config.InitConfig("config/config.toml")
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 启动服务器
	if err := server.Start(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
	
	errChan := make(chan error)

	select {
    case err := <-errChan:
        log.Fatalf("服务器错误: %v", err)
    case <-interruptChannel():  // 添加优雅退出
        log.Println("接收到中断信号，关闭服务器...")
    }
}

func interruptChannel() <-chan os.Signal {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    return c
}