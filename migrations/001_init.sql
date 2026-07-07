-- +goose Up
CREATE TABLE IF NOT EXISTS rooms (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_name  TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_room_created ON messages(room_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS rooms;
