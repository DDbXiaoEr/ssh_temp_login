BIN := build/ssh-sign
CONFIG ?= backend/config.yaml
ADDR ?= https://127.0.0.1:8080

.PHONY: all build frontend backend run open clean

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

build: frontend backend

run: build
	./$(BIN) -config "$(CONFIG)"

open: build
	(sleep 1 && python3 -c 'import webbrowser; webbrowser.open("$(ADDR)")') &
	./$(BIN) -config "$(CONFIG)"

clean:
	rm -f $(BIN)
