.PHONY: build test clean install lint fmt vet help

# Build variables
BINARY_NAME=aspec
BINARY_DIR=bin
CMD_DIR=cmd/aspec
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "✓ Built $(BINARY_DIR)/$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "✓ Built all platform binaries"

test: build ## Run tests
	@echo "Running tests..."
	go test -v ./...
	@echo "Testing CLI with mock provider..."
	./$(BINARY_DIR)/$(BINARY_NAME) test \
		--spec meeting_summary \
		--fixture testdata/meeting_summary/input.md \
		--expected testdata/meeting_summary/expected.json \
		--provider mock \
		--mock-fixture testdata/meeting_summary/mock_response.json
	@echo "✓ All tests passed"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v ./internal/...
	@echo "✓ Unit tests passed"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping lint check"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ go vet passed"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	go clean
	@echo "✓ Cleaned"

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./$(CMD_DIR)
	@echo "✓ Installed $(BINARY_NAME)"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

check: fmt vet lint test-unit ## Run all checks (format, vet, lint, unit tests)

ci: deps check build ## Run CI pipeline (deps, checks, build)

demo: build ## Run a quick demo
	@echo "Running demo..."
	@echo "1. Initializing config..."
	-./$(BINARY_DIR)/$(BINARY_NAME) init 2>/dev/null || true
	@echo "2. Showing version..."
	./$(BINARY_DIR)/$(BINARY_NAME) version
	@echo "3. Testing with mock provider..."
	./$(BINARY_DIR)/$(BINARY_NAME) test \
		--spec meeting_summary \
		--fixture testdata/meeting_summary/input.md \
		--provider mock \
		--mock-fixture testdata/meeting_summary/mock_response.json
	@echo "✓ Demo completed"

# Development helpers
dev-run: build ## Build and run with sample args
	./$(BINARY_DIR)/$(BINARY_NAME) list --help

dev-extract: build ## Test extract with sample data
	@echo "Testing extraction with sample meeting notes..."
	./$(BINARY_DIR)/$(BINARY_NAME) extract \
		--spec-path ../specs/artifacts/meeting_summary.schema.json \
		--in testdata/meeting_summary/input.md \
		--no-validate \
		|| echo "Note: This will fail without an API key - use for development only"

# Release helpers
release-prep: clean deps check build-all ## Prepare for release
	@echo "Release preparation complete"
	@echo "Binaries built:"
	@ls -la $(BINARY_DIR)/

.DEFAULT_GOAL := help