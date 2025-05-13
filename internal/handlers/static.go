package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"go-chatroom/internal/config"
)

//go:embed index.html
var staticFiles embed.FS

func InitStaticDir() error {
	staticDir := "static"
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		return fmt.Errorf("无法创建static目录: %v", err)
	}

	err = fs.WalkDir(staticFiles, "assets", func(path string, d fs.DirEntry, err error) error {
		uploadsDir := "uploads"
		err = os.MkdirAll(uploadsDir, 0755)
		if err != nil {
			return fmt.Errorf("无法创建static目录: %v", err)
		}
		
		return nil
	})
	return nil
}

func RenderIndex(w http.ResponseWriter, ui config.UIConfig) {
	tmpl := template.Must(template.New("index.html").ParseFS(staticFiles, "index.html"))

	data := struct {
		Title     string
		PageTitle string
	}{
		Title:     ui.Title,
		PageTitle: ui.PageTitle,
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "无法渲染模板", http.StatusInternalServerError)
	}
}