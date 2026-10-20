BIN := build/ssh-sign
LINUX_BIN := build/ssh-sign-linux
CONFIG ?= backend/config.yaml
ADDR ?= https://127.0.0.1:8080
IMAGE ?= ssh-sign
GIT_HASH := $(shell git rev-parse --short HEAD)
LDFLAGS_DEBUG := -X main.ginMode=debug
LDFLAGS_RELEASE := -s -w -X main.ginMode=release

.PHONY: all build debug release frontend backend backend-debug backend-release backend-linux docker run open clean

all: debug

frontend:
	cd frontend && npm install
	cd frontend && npm run build
	rm -rf backend/web
	mkdir -p backend/web
	cp -R build/web/. backend/web/

backend-debug:
	mkdir -p build
	cd backend && go build -ldflags "$(LDFLAGS_DEBUG)" -o ../$(BIN) .

backend-release:
	mkdir -p build
	cd backend && go build -ldflags "$(LDFLAGS_RELEASE)" -o ../$(BIN) .

backend-linux:
	mkdir -p build
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS_RELEASE)" -o ../$(LINUX_BIN) .

backend: backend-debug
debug: frontend backend-debug
release: frontend backend-release
build: debug

docker: frontend backend-linux
	docker build -t $(IMAGE):$(GIT_HASH) .

run: debug
	./$(BIN) -config "$(CONFIG)"

open: debug
	(sleep 1 && python3 -c 'import webbrowser; webbrowser.open("$(ADDR)")') &
	./$(BIN) -config "$(CONFIG)"

clean:
	rm -f $(BIN) $(LINUX_BIN)
