package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/uright008/chatroom/internal/database"
	"github.com/uright008/chatroom/internal/models"
)

var (
	clients    = make(map[*websocket.Conn]string)
	userColors = []string{
		"#3366cc", "#dc3912", "#ff9900", "#109618", "#990099",
		"#0099c6", "#dd4477", "#66aa00", "#b82e2e", "#316395",
	}
)

func HandleConnections(w http.ResponseWriter, r *http.Request, db *database.Database, upgrader websocket.Upgrader, maxHistory int) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	
	// 为新用户分配颜色
	color := userColors[len(clients)%len(userColors)]
	clients[ws] = color
	
	// 发送历史消息
	messages, err := db.GetHistoryMessages(maxHistory)
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
		var msg models.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("错误: %v", err)
			delete(clients, ws)
			break
		}
		
		msg.Time = time.Now()
		msg.Color = clients[ws]
		
		// 保存消息到数据库
		err = db.SaveMessage(msg)
		if err != nil {
			log.Println("保存消息失败:", err)
		}
		
		// 发送消息到广播通道
		broadcast <- msg
	}
}

func HandleMessages() {
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