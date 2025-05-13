package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go-chatroom/internal/database"
	"go-chatroom/internal/models"
)

func InitUploadDir(uploadDir string) error {
	return os.MkdirAll(uploadDir, 0755)
}

func HandleFileUpload(w http.ResponseWriter, r *http.Request, db *database.Database, uploadDir string, maxUploadSize int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单数据
	err := r.ParseMultipartForm(int64(maxUploadSize) << 20)
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

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	newFilename := uuid.New().String() + ext
	filePath := filepath.Join(uploadDir, newFilename)

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
	msg := models.Message{
		Username: username,
		Time:     time.Now(),
		FileURL:  "/uploads/" + newFilename,
		FileName: header.Filename,
		FileSize: fileInfo.Size(),
		IsFile:   true,
		Color:    "#3366cc", // 默认颜色，实际应该从客户端获取
	}

	// 保存消息到数据库
	err = db.SaveMessage(msg)
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