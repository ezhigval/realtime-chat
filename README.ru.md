# realtime-chat

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
[![CI](https://github.com/ezhigval/realtime-chat/actions/workflows/ci.yml/badge.svg)](https://github.com/ezhigval/realtime-chat/actions/workflows/ci.yml)
![License](https://img.shields.io/badge/license-MIT-blue)
![Tier](https://img.shields.io/badge/tier-middle-5319e7)

**[English](README.md)** · Русский

Чат по комнатам через WebSocket: история сообщений и онлайн-присутствие. Redis Pub/Sub связывает несколько инстансов сервера в общие комнаты.

## Быстрый старт

```bash
make docker-up   # postgres + redis + server1:8086 + server2:8087
curl -s -X POST localhost:8086/api/v1/rooms -H 'Content-Type: application/json' -d '{"name":"general"}' | jq
```

Подключите два клиента к **разным инстансам** — увидите доставку между ними:

```bash
# терминал A → инстанс 1
websocat "ws://localhost:8086/ws?room=1&user=alice"

# терминал B → инстанс 2
websocat "ws://localhost:8087/ws?room=1&user=bob"
```

Отправка JSON: `{"type":"message","content":"hello from alice"}`

## REST

| Метод | Путь | Описание |
|--------|------|----------|
| GET | `/api/v1/rooms` | список комнат |
| POST | `/api/v1/rooms` | создать комнату |
| GET | `/api/v1/rooms/{id}/messages?limit=50` | история (сначала новые) |
| GET | `/api/v1/rooms/{id}/presence` | кто онлайн |

## WebSocket

`GET /ws?room={id}&user={name}`

Входящие: `{"type":"message","content":"..."}`  
Исходящие: события `message`, `join`, `leave`, `presence`.

## Модель масштабирования

```
Client ──WS──► server1 ──Redis Pub/Sub──► server2 ──WS──► Client
                  │                            │
                  └──── PostgreSQL (history) ──┘
```

На каждом инстансе свой локальный hub; Redis передаёт события между нодами. Origin ID не даёт эхо на инстансе-издателе.

## Ограничения

- Без авторизации (демо-уровень)
- `CheckOrigin` принимает все origin — за gateway лучше ужесточить
- Presence TTL 60 с, heartbeat каждые 30 с

## Стек

Go 1.25 · chi · gorilla/websocket · PostgreSQL · Redis · [go-toolkit](https://github.com/ezhigval/go-toolkit)

Порт **8086** (второй инстанс **8087** в compose) · MIT
