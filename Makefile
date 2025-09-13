.PHONY: build clean install test lint run-login run-search run-status help

# Default target
all: build

# Build the application
build:
	go build -o moodify .

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/moodify-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/moodify-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/moodify-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/moodify-windows-amd64.exe .

# Install dependencies
deps:
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	rm -f moodify
	rm -rf dist/

# Install to system PATH
install: build
	sudo cp moodify /usr/local/bin/

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Quick development commands
run-login: build
	./moodify login

run-search: build
	./moodify search happy upbeat songs

run-status: build
	./moodify status

run-logout: build
	./moodify logout

# Development setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go mod tidy

# Check for security vulnerabilities
security:
	go list -json -deps ./... | nancy sleuth

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  build-all      - Build for multiple platforms"
	@echo "  deps           - Install/update dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install to system PATH"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Lint code"
	@echo "  fmt            - Format code"
	@echo "  run-login      - Build and run login command"
	@echo "  run-search     - Build and run example search"
	@echo "  run-status     - Build and run status check"
	@echo "  run-logout     - Build and run logout command"
	@echo "  dev-setup      - Set up development tools"
	@echo "  security       - Check for security vulnerabilities"
	@echo "  help           - Show this help message"
