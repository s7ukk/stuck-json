set -e

echo "=== stuck-json Toolchain Setup ==="

if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed. Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Detected Go version: $GO_VERSION"

echo "Syncing go.mod dependencies..."
go mod tidy

if ! command -v golangci-lint &> /dev/null; then
    echo "Auto-installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    
else
    echo "golangci-lint is already installed."
fi

echo "=== All dependencies and tools are up to date! ==="
