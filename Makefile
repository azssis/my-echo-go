REGISTRYSERVER?=docker.io
DOCKERIMAGE?=azssi/my-echo-go
VERSION=$(shell git describe --tags --abbrev=0)

GOBUILDVERSION=1.23.1
GITCOMMIT=$(shell git rev-parse HEAD)
PWD=$(shell pwd)

.PHONY: cert image push-image

CONTAINER_ENGINE?=podman
ifeq ($(shell command -v podman 2> /dev/null),)
  CONTAINER_ENGINE=docker
endif

cert:
	@echo "Generate certificate for: $(CN)"
	mkdir -p tls
	openssl req -x509 -newkey rsa:4096 -sha256 -nodes -keyout tls/server.key -out tls/server.crt -subj "/CN=$(CN)" -days 3650

image:
	@echo "build Version: $(VERSION), Git: $(GITCOMMIT), GoVersion: $(GOBUILDVERSION)"
	@$(CONTAINER_ENGINE) build . --platform linux/amd64 -f build/Dockerfile  \
            		--build-arg build_version=$(VERSION) \
            		--build-arg build_commit=$(GITCOMMIT) \
            		--build-arg go_base_version=$(GOBUILDVERSION) \
            		-t $(DOCKERIMAGE):$(VERSION)

push-image: image
	@$(CONTAINER_ENGINE) push azssi/my-echo-go
