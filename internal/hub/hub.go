package hub

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 4096
)

type Client struct {
	Hub      *Hub
	RoomID   int64
	UserName string
	Conn     *websocket.Conn
	Send     chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[int64]map[*Client]struct{}
	log     *slog.Logger
	onJoin  func(roomID int64, client *Client)
	onLeave func(roomID int64, client *Client)
}

func New(log *slog.Logger) *Hub {
	return &Hub{
		rooms: make(map[int64]map[*Client]struct{}),
		log:   log,
	}
}

func (h *Hub) SetCallbacks(onJoin, onLeave func(roomID int64, client *Client)) {
	h.onJoin = onJoin
	h.onLeave = onLeave
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[*Client]struct{})
	}
	h.rooms[client.RoomID][client] = struct{}{}
	h.mu.Unlock()

	if h.onJoin != nil {
		h.onJoin(client.RoomID, client)
	}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if clients, ok := h.rooms[client.RoomID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.Send)
		}
		if len(clients) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}
	h.mu.Unlock()

	if h.onLeave != nil {
		h.onLeave(client.RoomID, client)
	}
}

func (h *Hub) BroadcastLocal(roomID int64, msg model.WSOutbound) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("marshal broadcast", "error", err)
		return
	}
	h.BroadcastRaw(roomID, data)
}

func (h *Hub) BroadcastRaw(roomID int64, data []byte) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	snapshot := make([]*Client, 0, len(clients))
	for c := range clients {
		snapshot = append(snapshot, c)
	}
	h.mu.RUnlock()

	for _, c := range snapshot {
		select {
		case c.Send <- data:
		default:
			h.log.Warn("client send buffer full, dropping", "user", c.UserName)
		}
	}
}

func (c *Client) ReadPump(handle func(*Client, model.WSInbound)) {
	defer func() {
		c.Hub.Unregister(c)
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMsgSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var inbound model.WSInbound
		if err := c.Conn.ReadJSON(&inbound); err != nil {
			break
		}
		if handle != nil {
			handle(c, inbound)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
