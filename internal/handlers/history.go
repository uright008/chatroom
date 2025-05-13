package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/uright008/chatroom/internal/database"
)

func HandleHistoryRequest(w http.ResponseWriter, r *http.Request, db *database.Database, defaultLimit int) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	limitStr := r.URL.Query().Get("limit")
	limit := defaultLimit
	
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}
	
	messages, err := db.GetHistoryMessages(limit)
	if err != nil {
		http.Error(w, "无法获取历史消息", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}