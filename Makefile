REGISTRYSERVER?=docker.io
DOCKERIMAGE?=azssi/my-echo-go
VERSION=$(shell git describe --tags --abbrev=0)

GOBUILDVERSION=1.23.1
GITCOMMIT=$(shell git rev-parse HEAD)
PWD=$(shell pwd)

.PHONY: all build clean run run-dev help

CONTAINER_ENGINE?=podman
ifeq ($(shell command -v podman 2> /dev/null),)
  CONTAINER_ENGINE=docker
endif

# 기본 변수 설정
APP_NAME=echo-server
BUILD_DIR=bin
GO_FILES=$(shell find . -name "*.go")

# 기본 타겟
all: clean build

cert:
	@echo "Generate certificate for: $(CN)"
	mkdir -p tls
	openssl req -x509 -newkey rsa:4096 -sha256 -nodes -keyout tls/server.key -out tls/server.crt -subj "/CN=$(CN)" -days 3650

# 빌드 디렉터리 생성 및 Go 빌드
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# 빌드 결과물 삭제
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# 서버 실행
run: build
	@echo "Running $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME)

# 개발 모드로 실행 (상세 헤더 정보 표시)
run-dev: build
	@echo "Running $(APP_NAME) in development mode..."
	@ECHO_DETAILED_HEADER=true ECHO_PRETTY=true ./$(BUILD_DIR)/$(APP_NAME)

# 도움말
help:
	@echo "Available commands:"
	@echo "  make build      - Build the application"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make run        - Build and run the application"
	@echo "  make run-dev    - Run with detailed headers and pretty printing"
	@echo "  make help       - Show this help message"

image:
	@echo "build Version: $(VERSION), Git: $(GITCOMMIT), GoVersion: $(GOBUILDVERSION)"
	@$(CONTAINER_ENGINE) build . --platform linux/amd64 -f build/Dockerfile  \
            		--build-arg build_version=$(VERSION) \
            		--build-arg build_commit=$(GITCOMMIT) \
            		--build-arg go_base_version=$(GOBUILDVERSION) \
            		-t $(DOCKERIMAGE):$(VERSION)

push-image: image
	@$(CONTAINER_ENGINE) push azssi/my-echo-go
