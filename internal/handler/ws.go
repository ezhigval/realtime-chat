package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ezhigval/go-toolkit/httputil"
	"github.com/ezhigval/realtime-chat/internal/hub"
	"github.com/ezhigval/realtime-chat/internal/model"
	"github.com/ezhigval/realtime-chat/internal/service"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // demo; tighten in production
	},
}

type WS struct {
	svc *service.ChatService
	log *slog.Logger
}

func NewWS(svc *service.ChatService, log *slog.Logger) *WS {
	return &WS{svc: svc, log: log}
}

func (h *WS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(r.URL.Query().Get("room"), 10, 64)
	if err != nil || roomID <= 0 {
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "room query param required", err))
		return
	}
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	if user == "" {
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "user query param required", nil))
		return
	}

	ctx := r.Context()
	if _, err := h.svc.GetRoom(ctx, roomID); err != nil {
		status := http.StatusNotFound
		if err != service.ErrRoomNotFound {
			status = http.StatusInternalServerError
		}
		httputil.WriteError(w, httputil.NewAppError(status, "ROOM_ERROR", err.Error(), err))
		return
	}

	if err := h.svc.EnsureRoomSubscribed(ctx, roomID); err != nil {
		h.log.Error("subscribe room failed", "error", err, "room", roomID)
		httputil.WriteError(w, httputil.NewAppError(http.StatusInternalServerError, "SUBSCRIBE_FAILED", "redis subscribe failed", err))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("websocket upgrade failed", "error", err)
		return
	}

	client := &hub.Client{
		Hub:      nil,
		RoomID:   roomID,
		UserName: user,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	// Hub reference set via service internal hub - pass through service
	h.svc.RegisterClient(client)

	h.svc.SendJoinNotice(ctx, client)

	go client.WritePump()
	go h.heartbeatLoop(r.Context(), client)

	client.ReadPump(func(c *hub.Client, inbound model.WSInbound) {
		h.svc.HandleInbound(r.Context(), c, inbound)
	})
}

func (h *WS) heartbeatLoop(ctx context.Context, client *hub.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.svc.Heartbeat(ctx, client.RoomID, client.UserName)
		}
	}
}
