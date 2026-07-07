package presence

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 60 * time.Second
const keyPrefix = "chat:presence:"

type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) key(roomID int64) string {
	return keyPrefix + strconv.FormatInt(roomID, 10)
}

func (s *Store) Join(ctx context.Context, roomID int64, user string) error {
	pipe := s.rdb.Pipeline()
	memberKey := s.key(roomID)
	pipe.HSet(ctx, memberKey, user, time.Now().UTC().Unix())
	pipe.Expire(ctx, memberKey, ttl*2)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Leave(ctx context.Context, roomID int64, user string) error {
	return s.rdb.HDel(ctx, s.key(roomID), user).Err()
}

func (s *Store) Heartbeat(ctx context.Context, roomID int64, user string) error {
	return s.Join(ctx, roomID, user)
}

func (s *Store) List(ctx context.Context, roomID int64) ([]string, error) {
	members, err := s.rdb.HKeys(ctx, s.key(roomID)).Result()
	if err != nil {
		return nil, fmt.Errorf("presence list: %w", err)
	}
	if members == nil {
		return []string{}, nil
	}
	return members, nil
}
