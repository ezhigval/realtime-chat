package hub

import (
	"testing"

	"log/slog"

	"github.com/ezhigval/realtime-chat/internal/model"
)

func TestHub_BroadcastLocal(t *testing.T) {
	h := New(slog.Default())
	roomID := int64(1)

	received := make(chan []byte, 1)
	client := &Client{
		Hub:      h,
		RoomID:   roomID,
		UserName: "alice",
		Send:     received,
	}
	h.Register(client)

	h.BroadcastLocal(roomID, model.WSOutbound{
		Type:    model.WSTypeMessage,
		User:    "bob",
		Content: "hi",
	})

	select {
	case msg := <-received:
		if len(msg) == 0 {
			t.Fatal("empty message")
		}
	default:
		t.Fatal("expected broadcast")
	}
}

func TestHub_UnregisterClosesSend(t *testing.T) {
	h := New(slog.Default())
	client := &Client{
		Hub:      h,
		RoomID:   1,
		UserName: "alice",
		Send:     make(chan []byte, 1),
	}
	h.Register(client)
	h.Unregister(client)

	select {
	case _, ok := <-client.Send:
		if ok {
			t.Fatal("send channel should be closed")
		}
	default:
		t.Fatal("expected closed channel")
	}
}
