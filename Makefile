.PHONY: run test lint docker-up docker-down build migrate-up

DATABASE_URL ?= postgres://chat:chat@localhost:5436/chat?sslmode=disable

run:
	DATABASE_URL=$(DATABASE_URL) REDIS_ADDR=localhost:6384 go run ./cmd/server

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up
