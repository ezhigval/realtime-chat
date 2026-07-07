package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepository struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{pool: pool}
}

func (r *RoomRepository) Create(ctx context.Context, name string) (*model.Room, error) {
	var room model.Room
	err := r.pool.QueryRow(ctx, `
		INSERT INTO rooms (name) VALUES ($1)
		RETURNING id, name, created_at
	`, name).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) GetByID(ctx context.Context, id int64) (*model.Room, error) {
	var room model.Room
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, created_at FROM rooms WHERE id = $1
	`, id).Scan(&room.ID, &room.Name, &room.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) List(ctx context.Context) ([]model.Room, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, created_at FROM rooms ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		var room model.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}
