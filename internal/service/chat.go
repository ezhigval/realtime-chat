package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ezhigval/realtime-chat/internal/hub"
	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/ezhigval/realtime-chat/internal/presence"
	"github.com/ezhigval/realtime-chat/internal/pubsub"
	"github.com/ezhigval/realtime-chat/internal/repository"
)

var (
	ErrRoomNotFound   = errors.New("room not found")
	ErrEmptyName      = errors.New("room name required")
	ErrEmptyMessage   = errors.New("message content required")
	ErrEmptyUser      = errors.New("user name required")
)

type ChatService struct {
	rooms    *repository.RoomRepository
	messages *repository.MessageRepository
	presence *presence.Store
	bus      *pubsub.Bus
	hub      *hub.Hub
}

func NewChatService(
	rooms *repository.RoomRepository,
	messages *repository.MessageRepository,
	presence *presence.Store,
	bus *pubsub.Bus,
	h *hub.Hub,
) *ChatService {
	s := &ChatService{
		rooms:    rooms,
		messages: messages,
		presence: presence,
		bus:      bus,
		hub:      h,
	}

	bus.SetHandler(func(roomID int64, msg model.WSOutbound) {
		h.BroadcastLocal(roomID, msg)
	})

	h.SetCallbacks(s.onClientJoin, s.onClientLeave)
	return s
}

func (s *ChatService) CreateRoom(ctx context.Context, name string) (*model.Room, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	return s.rooms.Create(ctx, name)
}

func (s *ChatService) ListRooms(ctx context.Context) ([]model.Room, error) {
	rooms, err := s.rooms.List(ctx)
	if err != nil {
		return nil, err
	}
	if rooms == nil {
		return []model.Room{}, nil
	}
	return rooms, nil
}

func (s *ChatService) GetRoom(ctx context.Context, id int64) (*model.Room, error) {
	room, err := s.rooms.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

func (s *ChatService) ListMessages(ctx context.Context, roomID int64, limit int, before time.Time) ([]model.Message, error) {
	if _, err := s.GetRoom(ctx, roomID); err != nil {
		return nil, err
	}
	msgs, err := s.messages.List(ctx, roomID, limit, before)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		return []model.Message{}, nil
	}
	return msgs, nil
}

func (s *ChatService) ListPresence(ctx context.Context, roomID int64) ([]string, error) {
	if _, err := s.GetRoom(ctx, roomID); err != nil {
		return nil, err
	}
	return s.presence.List(ctx, roomID)
}

func (s *ChatService) RegisterClient(client *hub.Client) {
	client.Hub = s.hub
	s.hub.Register(client)
}

func (s *ChatService) UnregisterClient(client *hub.Client) {
	s.hub.Unregister(client)
}

func (s *ChatService) EnsureRoomSubscribed(ctx context.Context, roomID int64) error {
	return s.bus.SubscribeRoom(ctx, roomID)
}

func (s *ChatService) onClientJoin(roomID int64, client *hub.Client) {
	ctx := context.Background()
	_ = s.presence.Join(ctx, roomID, client.UserName)
	s.broadcastPresence(ctx, roomID)
}

func (s *ChatService) onClientLeave(roomID int64, client *hub.Client) {
	ctx := context.Background()
	_ = s.presence.Leave(ctx, roomID, client.UserName)
	s.broadcastPresence(ctx, roomID)

	out := model.WSOutbound{
		Type:   model.WSTypeLeave,
		User:   client.UserName,
		RoomID: roomID,
		At:     time.Now().UTC(),
	}
	s.hub.BroadcastLocal(roomID, out)
	_ = s.bus.Publish(ctx, roomID, out)
}

func (s *ChatService) HandleInbound(ctx context.Context, client *hub.Client, inbound model.WSInbound) {
	switch inbound.Type {
	case model.WSTypeMessage, "":
		s.handleMessage(ctx, client, inbound.Content)
	default:
		return
	}
}

func (s *ChatService) handleMessage(ctx context.Context, client *hub.Client, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	msg, err := s.messages.Insert(ctx, client.RoomID, client.UserName, content)
	if err != nil {
		return
	}

	out := model.WSOutbound{
		Type:      model.WSTypeMessage,
		User:      client.UserName,
		Content:   msg.Content,
		MessageID: msg.ID,
		RoomID:    client.RoomID,
		At:        msg.CreatedAt,
		Origin:    s.bus.InstanceID(),
	}

	s.hub.BroadcastLocal(client.RoomID, out)
	_ = s.bus.Publish(ctx, client.RoomID, out)
}

func (s *ChatService) broadcastPresence(ctx context.Context, roomID int64) {
	users, err := s.presence.List(ctx, roomID)
	if err != nil {
		return
	}
	out := model.WSOutbound{
		Type:   model.WSTypePresence,
		RoomID: roomID,
		Users:  users,
		At:     time.Now().UTC(),
	}
	s.hub.BroadcastLocal(roomID, out)
	_ = s.bus.Publish(ctx, roomID, out)
}

func (s *ChatService) SendJoinNotice(ctx context.Context, client *hub.Client) {
	out := model.WSOutbound{
		Type:   model.WSTypeJoin,
		User:   client.UserName,
		RoomID: client.RoomID,
		At:     time.Now().UTC(),
	}
	s.hub.BroadcastLocal(client.RoomID, out)
	_ = s.bus.Publish(ctx, client.RoomID, out)
}

func (s *ChatService) Heartbeat(ctx context.Context, roomID int64, user string) {
	_ = s.presence.Heartbeat(ctx, roomID, user)
}
