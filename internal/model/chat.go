package model

import "time"

type Room struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	UserName  string    `json:"user"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type WSInbound struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

type WSOutbound struct {
	Type      string    `json:"type"`
	User      string    `json:"user,omitempty"`
	Content   string    `json:"content,omitempty"`
	MessageID int64     `json:"message_id,omitempty"`
	RoomID    int64     `json:"room_id,omitempty"`
	Users     []string  `json:"users,omitempty"`
	At        time.Time `json:"at,omitempty"`
	Origin    string    `json:"origin,omitempty"`
}

const (
	WSTypeMessage = "message"
	WSTypeJoin    = "join"
	WSTypeLeave   = "leave"
	WSTypePresence = "presence"
)
