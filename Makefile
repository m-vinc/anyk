APP_NAME := anyk
VERSION := 1.0.0
BUILD_DIR := ./build
BUILD_FLAGS :=
BUILD_TYPE := development

.PHONY: build clean help

build:
	@echo "Building $(APP_NAME) with version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -trimpath -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

