.PHONY: run test test-integration migrate-up migrate-down

run:
	go run cmd/server/main.go

test:
	go test -v -short ./...

test-integration:
	go test -v -tags=integration ./...

migrate-up:
	migrate -path migrations -database "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable" down
