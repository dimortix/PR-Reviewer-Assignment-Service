.PHONY: build run test test-coverage clean docker-up docker-down docker-restart lint fmt deps

build:
	go build -o bin/server cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down -v

docker-restart:
	docker-compose down
	docker-compose up --build

migrate-up:
	migrate -path migrations -database "postgresql://user:password@localhost:5432/pr_reviewer?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgresql://user:password@localhost:5432/pr_reviewer?sslmode=disable" down

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	goimports -w .

deps:
	go mod download
	go mod verify
	go mod tidy
