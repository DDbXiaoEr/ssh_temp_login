BIN := build/ssh-sign
LINUX_BIN := build/ssh-sign-linux
CONFIG ?= backend/config.yaml
ADDR ?= https://127.0.0.1:8080
IMAGE ?= ssh-sign
GIT_HASH := $(shell git rev-parse --short HEAD)

.PHONY: all build frontend backend backend-linux docker run open clean

all: build

frontend:
	cd frontend && npm install
	cd frontend && npm run build
	rm -rf backend/web
	mkdir -p backend/web
	cp -R build/web/. backend/web/

backend:
	mkdir -p build
	cd backend && go build -o ../build/ssh-sign .

backend-linux:
	mkdir -p build
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../build/ssh-sign-linux .

build: frontend backend

docker: frontend backend-linux
	docker build -t $(IMAGE):$(GIT_HASH) .

run: build
	./$(BIN) -config "$(CONFIG)"

open: build
	(sleep 1 && python3 -c 'import webbrowser; webbrowser.open("$(ADDR)")') &
	./$(BIN) -config "$(CONFIG)"

clean:
	rm -f $(BIN) $(LINUX_BIN)
