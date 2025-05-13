package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"go-chatroom/internal/config"
)

//go:embed assets/*
var staticFiles embed.FS

func InitStaticDir() error {
	staticDir := "static"
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		return fmt.Errorf("无法创建static目录: %v", err)
	}

	err = fs.WalkDir(staticFiles, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "assets" {
			return nil
		}

		// Calculate the target path
		relPath, err := filepath.Rel("assets", path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(staticDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		 }

		// Read the embedded file
		data, err := staticFiles.ReadFile(path)
		if err != nil {
			return err
		}

		// Write the target file
		return os.WriteFile(targetPath, data, 0644)
	})

	if err != nil {
		return fmt.Errorf("无法提取静态文件: %v", err)
	}

	return nil
}

func RenderIndex(w http.ResponseWriter, ui config.UIConfig) {
	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, "无法加载模板", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title     string
		PageTitle string
	}{
		Title:     ui.Title,
		PageTitle: ui.PageTitle,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "无法渲染模板", http.StatusInternalServerError)
	}
}