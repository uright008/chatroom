package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/uright008/chatroom/internal/config"
	"github.com/uright008/chatroom/internal/models"
)

type Database struct {
	db *sql.DB
}

func New(cfg *config.DatabaseConfig) (*Database, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := initTables(db); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func initTables(db *sql.DB) error {
	_, err := db.Exec(`
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
	return err
}

func (d *Database) SaveMessage(msg models.Message) error {
	_, err := d.db.Exec(`
		INSERT INTO messages 
		(username, text, time, color, file_url, file_name, file_size, is_file)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.Username, msg.Text, msg.Time, msg.Color, 
		msg.FileURL, msg.FileName, msg.FileSize, msg.IsFile)
	return err
}

func (d *Database) GetHistoryMessages(limit int) ([]models.Message, error) {
	rows, err := d.db.Query(`
		SELECT username, text, time, color, file_url, file_name, file_size, is_file
		FROM messages 
		ORDER BY id DESC 
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
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