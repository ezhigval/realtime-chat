package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"

	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/redis/go-redis/v9"
)

const channelPrefix = "chat:room:"

type Bus struct {
	rdb        *redis.Client
	instanceID string
	log        *slog.Logger
	onMessage  func(roomID int64, msg model.WSOutbound)

	mu       sync.Mutex
	subs     map[int64]*redis.PubSub
	cancelFn map[int64]context.CancelFunc
}

func NewBus(rdb *redis.Client, instanceID string, log *slog.Logger) *Bus {
	return &Bus{
		rdb:        rdb,
		instanceID: instanceID,
		log:        log,
		subs:       make(map[int64]*redis.PubSub),
		cancelFn:   make(map[int64]context.CancelFunc),
	}
}

func (b *Bus) SetHandler(fn func(roomID int64, msg model.WSOutbound)) {
	b.onMessage = fn
}

func (b *Bus) channel(roomID int64) string {
	return channelPrefix + strconv.FormatInt(roomID, 10)
}

func (b *Bus) Publish(ctx context.Context, roomID int64, msg model.WSOutbound) error {
	msg.Origin = b.instanceID
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return b.rdb.Publish(ctx, b.channel(roomID), data).Err()
}

func (b *Bus) SubscribeRoom(ctx context.Context, roomID int64) error {
	b.mu.Lock()
	if _, ok := b.subs[roomID]; ok {
		b.mu.Unlock()
		return nil
	}

	subCtx, cancel := context.WithCancel(ctx)
	ps := b.rdb.Subscribe(subCtx, b.channel(roomID))
	b.subs[roomID] = ps
	b.cancelFn[roomID] = cancel
	b.mu.Unlock()

	go b.listen(subCtx, roomID, ps)
	return nil
}

func (b *Bus) listen(ctx context.Context, roomID int64, ps *redis.PubSub) {
	ch := ps.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var out model.WSOutbound
			if err := json.Unmarshal([]byte(msg.Payload), &out); err != nil {
				b.log.Error("pubsub unmarshal", "error", err)
				continue
			}
			if out.Origin == b.instanceID {
				continue
			}
			if b.onMessage != nil {
				b.onMessage(roomID, out)
			}
		}
	}
}

func (b *Bus) UnsubscribeRoom(roomID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if cancel, ok := b.cancelFn[roomID]; ok {
		cancel()
		delete(b.cancelFn, roomID)
	}
	delete(b.subs, roomID)
}

func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, cancel := range b.cancelFn {
		cancel()
		delete(b.cancelFn, id)
	}
	b.subs = make(map[int64]*redis.PubSub)
	return nil
}

func (b *Bus) InstanceID() string {
	return b.instanceID
}

func RoomChannel(roomID int64) string {
	return fmt.Sprintf("%s%d", channelPrefix, roomID)
}
