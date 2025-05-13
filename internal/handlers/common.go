package handlers

import (
	"go-chatroom/internal/models"
)

var (
	broadcast = make(chan models.Message)
)