package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

// 配置结构体
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	UI       UIConfig
}

type UIConfig struct {
	Title      string `toml:"title"`
	PageTitle  string `toml:"page_title"`
}

type ServerConfig struct {
	Port           string `toml:"port"`
	MaxHistory     int    `toml:"max_history"`
	UploadDir      string `toml:"upload_dir"`
	MaxUploadSize  int    `toml:"max_upload_size"`
}

type DatabaseConfig struct {
	Driver string `toml:"driver"`
	DSN    string `toml:"dsn"`
}

// 消息结构体
type Message struct {
	ID        int       `json:"-"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Time      time.Time `json:"time"`
	Color     string    `json:"color"`
	FileURL   string    `json:"file_url,omitempty"`
	FileName  string    `json:"file_name,omitempty"`
	FileSize  int64     `json:"file_size,omitempty"`
	IsFile    bool      `json:"is_file"`
}

var (
	config     Config
	db         *sql.DB
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients    = make(map[*websocket.Conn]string) // 客户端连接与颜色映射
	broadcast  = make(chan Message)
	userColors = []string{
		"#3366cc", "#dc3912", "#ff9900", "#109618", "#990099",
		"#0099c6", "#dd4477", "#66aa00", "#b82e2e", "#316395",
	}
)

func initConfig() {
	configPath := "config/config.toml"
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll("config", 0755)
		if err != nil {
			log.Fatal("无法创建配置目录:", err)
		}

		config = Config{
			Server: ServerConfig{
				Port:          ":8080",
				MaxHistory:    100,
				UploadDir:     "./static/uploads",
				MaxUploadSize: 10,
			},
			Database: DatabaseConfig{
				Driver: "sqlite",
				DSN:    "./chatroom.db",
			},
			UI: UIConfig{
				Title:      "我的聊天室",
				PageTitle:  "欢迎来到我的聊天室",
			},
		}

		f, err := os.Create(configPath)
		if err != nil {
			log.Fatal("无法创建配置文件:", err)
		}
		defer f.Close()

		encoder := toml.NewEncoder(f)
		err = encoder.Encode(config)
		if err != nil {
			log.Fatal("无法编码配置:", err)
		}
	} else {
		_, err := toml.DecodeFile(configPath, &config)
		if err != nil {
			log.Fatal("无法解析配置文件:", err)
		}
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", config.Database.DSN)
	if err != nil {
		log.Fatal("无法连接数据库:", err)
	}

	// 创建消息表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			text TEXT,
			time DATETIME,
			color TEXT,
			file_url TEXT,
			file_name TEXT,
			file_size INTEGER,
			is_file BOOLEAN
		)
	`)
	if err != nil {
		log.Fatal("创建表失败:", err)
	}
}

func initUploadDir() {
	err := os.MkdirAll(config.Server.UploadDir, 0755)
	if err != nil {
		log.Fatal("无法创建上传目录:", err)
	}
}

func saveMessage(msg Message) error {
	_, err := db.Exec(`
		INSERT INTO messages 
		(username, text, time, color, file_url, file_name, file_size, is_file)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.Username, msg.Text, msg.Time, msg.Color, 
		msg.FileURL, msg.FileName, msg.FileSize, msg.IsFile)
	return err
}

func getHistoryMessages(limit int) ([]Message, error) {
	rows, err := db.Query(`
		SELECT username, text, time, color, file_url, file_name, file_size, is_file
		FROM messages 
		ORDER BY id DESC 
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.Username, &msg.Text, &msg.Time, &msg.Color,
			&msg.FileURL, &msg.FileName, &msg.FileSize, &msg.IsFile)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// 反转消息顺序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	
	// 为新用户分配颜色
	color := userColors[len(clients)%len(userColors)]
	clients[ws] = color
	
	// 发送历史消息
	messages, err := getHistoryMessages(config.Server.MaxHistory)
	if err != nil {
		log.Println("获取历史消息失败:", err)
	}
	
	for _, msg := range messages {
		err := ws.WriteJSON(msg)
		if err != nil {
			log.Printf("错误: %v", err)
			delete(clients, ws)
			return
		}
	}
	
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("错误: %v", err)
			delete(clients, ws)
			break
		}
		
		msg.Time = time.Now()
		msg.Color = clients[ws]
		
		// 保存消息到数据库
		err = saveMessage(msg)
		if err != nil {
			log.Println("保存消息失败:", err)
		}
		
		// 发送消息到广播通道
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("错误: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单数据，限制内存使用
	err := r.ParseMultipartForm(int64(config.Server.MaxUploadSize) << 20) // MB to bytes
	if err != nil {
		http.Error(w, "无法解析表单数据: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "无法获取上传文件: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 获取用户名
	username := r.FormValue("username")
	if username == "" {
		username = "匿名用户"
	}

	// 确保上传目录存在
	err = os.MkdirAll(config.Server.UploadDir, 0755)
	if err != nil {
		http.Error(w, "无法创建上传目录: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	newFilename := uuid.New().String() + ext
	filePath := filepath.Join(config.Server.UploadDir, newFilename)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "无法创建目标文件: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 复制文件内容
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "无法保存文件内容: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取文件信息
	fileInfo, err := dst.Stat()
	if err != nil {
		http.Error(w, "无法获取文件信息: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建消息
	msg := Message{
		Username: username,
		Time:     time.Now(),
		FileURL:  "/uploads/" + newFilename,
		FileName: header.Filename,
		FileSize: fileInfo.Size(),
		IsFile:   true,
		Color:    "#3366cc", // 默认颜色，实际应该从客户端获取
	}

	// 保存消息到数据库
	err = saveMessage(msg)
	if err != nil {
		http.Error(w, "无法保存消息到数据库: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 广播消息
	broadcast <- msg

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func handleHistoryRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	limitStr := r.URL.Query().Get("limit")
	limit := config.Server.MaxHistory
	
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}
	
	messages, err := getHistoryMessages(limit)
	if err != nil {
		http.Error(w, "无法获取历史消息", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func renderIndex(w http.ResponseWriter) {
	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, "无法加载模板", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title     string
		PageTitle string
	}{
		Title:     config.UI.Title,
		PageTitle: config.UI.PageTitle,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "无法渲染模板", http.StatusInternalServerError)
	}
}

var staticFiles embed.FS

func initStaticDir() error {
	// 从嵌入的文件系统中提取静态文件
	staticDir := "static"
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		return fmt.Errorf("无法创建static目录: %v", err)
	}

	// 遍历嵌入的静态文件并写入磁盘
	err = fs.WalkDir(staticFiles, "assets/static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录
		if path == "assets/static" {
			return nil
		}

		// 计算目标路径
		relPath, err := filepath.Rel("assets/static", path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(staticDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 读取嵌入的文件
		data, err := staticFiles.ReadFile(path)
		if err != nil {
			return err
		}

		// 写入目标文件
		return os.WriteFile(targetPath, data, 0644)
	})

	if err != nil {
		return fmt.Errorf("无法提取静态文件: %v", err)
	}

	return nil
}

func main() {
	flag.Parse()
	
	// 初始化静态文件目录
	if err := initStaticDir(); err != nil {
		log.Fatalf("初始化静态文件失败: %v", err)
	}

	// 初始化配置和数据库
	initConfig()
	initDB()
	initUploadDir()
	
	// 静态文件服务
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			renderIndex(w)
			return
		}
		http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
	})
	
	// WebSocket路由
	http.HandleFunc("/ws", handleConnections)
	
	// 文件上传路由
	http.HandleFunc("/upload", handleFileUpload)
	
	// 历史消息路由
	http.HandleFunc("/history", handleHistoryRequest)
	
	// 启动消息广播
	go handleMessages()
	
	log.Println("服务器启动，监听", config.Server.Port)
	err := http.ListenAndServe(config.Server.Port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}