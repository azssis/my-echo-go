REGISTRYSERVER?=docker.io
DOCKERIMAGE?=azssi/my-echo-go

PWD=$(shell pwd)
GO_VERSION=1.23.1

GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_COMMIT_SHORT=$(shell git rev-parse --short=10 HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG=$(shell git describe --tags --exact-match 2>/dev/null || true)

ifeq ($(GIT_BRANCH),main)
    ifeq ($(GIT_TAG),)
        VERSION=$(GIT_COMMIT_SHORT)
    else
        VERSION=$(GIT_TAG)
    endif
else
    GIT_SAFE_BRANCH=$(shell echo $(GIT_BRANCH) | tr / _ | tr -d '\#[:space:]'  )
    VERSION=$(GIT_SAFE_BRANCH)
endif

.PHONY: all build clean run run-dev help docker-build docker-push

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
	@LOG_LEVEL=TRACE ECHO_LEVEL=TRACE ECHO_DETAILED_HEADER=true ECHO_PRETTY=true ./$(BUILD_DIR)/$(APP_NAME)

image:
	@echo "Building docker image... (Version: $(VERSION), Git: $(GIT_COMMIT), GoVersion: $(GO_VERSION))"
	@$(CONTAINER_ENGINE) build . --platform linux/amd64 -f build/Dockerfile \
		--build-arg build_version=$(VERSION) \
		--build-arg build_commit=$(GIT_COMMIT) \
		--build-arg go_base_version=$(GO_VERSION) \
		-t $(DOCKERIMAGE):$(VERSION)

# 도커 이미지 푸시
push-image: image
	@echo "Pushing docker image..."
	@$(CONTAINER_ENGINE) tag $(DOCKERIMAGE):$(VERSION) $(REGISTRYSERVER)/$(DOCKERIMAGE):$(VERSION)
	@$(CONTAINER_ENGINE) push $(REGISTRYSERVER)/$(DOCKERIMAGE):$(VERSION)
	@$(CONTAINER_ENGINE) rmi $(REGISTRYSERVER)/$(DOCKERIMAGE):$(VERSION)

# 도움말
help:
	@echo "Available commands:"
	@echo "  make build      - Build the application"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make run        - Build and run the application"
	@echo "  make run-dev    - Run with detailed headers and pretty printing"
	@echo "  make image      - Build docker image"
	@echo "  make push-image - Push docker image to registry"
	@echo "  make help       - Show this help message"
