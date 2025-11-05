.PHONY: run build test clean

APP=backend-woltapp-completion
BIN=dopc

run:
	go run ./cmd/$(BIN)

build:
	go build -o $(BIN) ./cmd/$(BIN)

test:
	go test ./...

clean:
	rm -f $(BIN)

