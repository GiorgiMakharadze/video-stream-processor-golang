BINARY=./bin/video-stream-processor-golang

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server

run: build
	$(BINARY)

run-dev:
	go run ./cmd/server/main.go

clean:
	rm -f $(BINARY)

test:
	go test -v ./...

format:
	go fmt ./...

.PHONY: build run clean test format
