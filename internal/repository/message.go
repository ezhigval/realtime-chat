package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Insert(ctx context.Context, roomID int64, user, content string) (*model.Message, error) {
	var msg model.Message
	err := r.pool.QueryRow(ctx, `
		INSERT INTO messages (room_id, user_name, content)
		VALUES ($1, $2, $3)
		RETURNING id, room_id, user_name, content, created_at
	`, roomID, user, content).Scan(&msg.ID, &msg.RoomID, &msg.UserName, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}
	return &msg, nil
}

func (r *MessageRepository) List(ctx context.Context, roomID int64, limit int, before time.Time) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows pgxRows
	var err error
	if before.IsZero() {
		rows, err = r.pool.Query(ctx, `
			SELECT id, room_id, user_name, content, created_at
			FROM messages WHERE room_id = $1
			ORDER BY created_at DESC LIMIT $2
		`, roomID, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, room_id, user_name, content, created_at
			FROM messages WHERE room_id = $1 AND created_at < $2
			ORDER BY created_at DESC LIMIT $3
		`, roomID, before, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.UserName, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

type pgxRows interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}
