# realtime-chat

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
[![CI](https://github.com/ezhigval/realtime-chat/actions/workflows/ci.yml/badge.svg)](https://github.com/ezhigval/realtime-chat/actions/workflows/ci.yml)
![License](https://img.shields.io/badge/license-MIT-blue)
![Tier](https://img.shields.io/badge/tier-middle-5319e7)

**English** · [Русский](README.ru.md)

Room-based WebSocket chat with message history and online presence. Redis Pub/Sub lets multiple server instances share the same rooms.

## Quick start

```bash
make docker-up   # postgres + redis + server1:8086 + server2:8087
curl -s -X POST localhost:8086/api/v1/rooms -H 'Content-Type: application/json' -d '{"name":"general"}' | jq
```

Connect two clients on **different instances** to see cross-instance delivery:

```bash
# terminal A → instance 1
websocat "ws://localhost:8086/ws?room=1&user=alice"

# terminal B → instance 2
websocat "ws://localhost:8087/ws?room=1&user=bob"
```

Send JSON: `{"type":"message","content":"hello from alice"}`

## REST

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/rooms` | list rooms |
| POST | `/api/v1/rooms` | create room |
| GET | `/api/v1/rooms/{id}/messages?limit=50` | history (newest first) |
| GET | `/api/v1/rooms/{id}/presence` | online users |

## WebSocket

`GET /ws?room={id}&user={name}`

Inbound: `{"type":"message","content":"..."}`  
Outbound: `message`, `join`, `leave`, `presence` events.

## Scaling model

```
Client ──WS──► server1 ──Redis Pub/Sub──► server2 ──WS──► Client
                  │                            │
                  └──── PostgreSQL (history) ──┘
```

Each instance runs a local hub; Redis carries events between nodes. Origin ID skips echo on the publisher instance.

## Known limits

- No auth (demo scope)
- `CheckOrigin` accepts all origins — tighten behind your gateway
- Presence TTL 60s with heartbeat every 30s

## Stack

Go 1.25 · chi · gorilla/websocket · PostgreSQL · Redis · [go-toolkit](https://github.com/ezhigval/go-toolkit)

Port **8086** (second instance **8087** in compose) · MIT
