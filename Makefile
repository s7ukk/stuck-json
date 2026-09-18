.PHONY: test bench tidy lint setup help

# Default target
all: test

## setup: Auto-installs all required development tools (golangci-lint, govulncheck)
setup:
	@echo "Checking and installing Go dev tools..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	@go mod tidy
	@echo "All tools are ready."

## test: Runs unit tests with race detector and coverage
test:
	go test -v -race -cover ./...

## bench: Runs performance benchmarks
bench:
	go test -benchmem -bench=. .

## tidy: Ensures go.mod and go.sum are clean
tidy:
	go mod tidy
	go mod verify

## lint: Runs code analysis
lint:
	golangci-lint run ./...

## help: Display available make tasks
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
