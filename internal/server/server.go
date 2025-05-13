package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"go-chatroom/internal/config"
	"go-chatroom/internal/database"
	"go-chatroom/internal/handlers"
)

type Server struct {
	cfg      *config.Config
	db       *database.Database
	upgrader websocket.Upgrader
}

func Start(cfg *config.Config) error {
	// 初始化数据库
	db, err := database.New(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// 初始化上传目录
	if err := handlers.InitUploadDir(cfg.Server.UploadDir); err != nil {
		return fmt.Errorf("failed to initialize upload directory: %w", err)
	}

	// 初始化静态文件
	if err := handlers.InitStaticDir(); err != nil {
		return fmt.Errorf("failed to initialize static files: %w", err)
	}

	// 创建服务器实例
	srv := &Server{
		cfg: cfg,
		db:  db,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	// 设置路由
	http.HandleFunc("/", srv.handleStatic)
	http.HandleFunc("/ws", srv.handleWebSocket)
	http.HandleFunc("/upload", srv.handleFileUpload)
	http.HandleFunc("/history", srv.handleHistoryRequest)

	// 启动消息广播
	go handlers.HandleMessages()

	log.Println("服务器启动，监听", cfg.Server.Port)
	return http.ListenAndServe(cfg.Server.Port, nil)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		handlers.RenderIndex(w, s.cfg.UI)
		return
	}
	http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	handlers.HandleConnections(w, r, s.db, s.upgrader, s.cfg.Server.MaxHistory)
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	handlers.HandleFileUpload(w, r, s.db, s.cfg.Server.UploadDir, s.cfg.Server.MaxUploadSize)
}

func (s *Server) handleHistoryRequest(w http.ResponseWriter, r *http.Request) {
	handlers.HandleHistoryRequest(w, r, s.db, s.cfg.Server.MaxHistory)
}