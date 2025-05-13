package main

import (
	"log"

	"github.com/uright008/chatroom/internal/config"
	"github.com/uright008/chatroom/internal/server"
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
}