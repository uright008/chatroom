package models

import "time"

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