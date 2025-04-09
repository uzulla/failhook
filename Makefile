.PHONY: all build clean test

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=failhook
BIN_DIR=bin

all: test build

build: build-darwin-arm64 build-linux-amd64 build-linux-arm64

build-darwin-arm64:
	@echo "Building for macOS (Apple Silicon)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 -v

build-linux-amd64:
	@echo "Building for Linux (AMD64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 -v

build-linux-arm64:
	@echo "Building for Linux (ARM64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 -v

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	$(GOCLEAN)

# Cross compilation
.PHONY: build-darwin-arm64 build-linux-amd64 build-linux-arm64