GO := go
APP := ./cmd/server

.PHONY: build test run docker-up docker-down

build:
	$(GO) build ./...

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/server

docker-up:
	docker compose -f docker-compose.prod.yml up --build -d

docker-down:
	docker compose -f docker-compose.prod.yml down -v
