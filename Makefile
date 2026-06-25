# GoTypeMyAdmin — build & run helpers.

BACKEND := backend
FRONTEND := frontend
BIN := bin/gotypemyadmin
ADDR ?= :8088

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install frontend deps
	cd $(FRONTEND) && npm install

.PHONY: dev-frontend
dev-frontend: ## Run Vite dev server (proxies /api → :8088)
	cd $(FRONTEND) && npm run dev

.PHONY: dev-backend
dev-backend: ## Run the Go API server (serves ../frontend/dist)
	cd $(BACKEND) && go run . -addr $(ADDR)

.PHONY: build-frontend
build-frontend: ## Type-check + bundle the frontend
	cd $(FRONTEND) && npm run build

.PHONY: build
build: build-frontend ## Build the single production binary (embeds nothing; serves dist/)
	cd $(BACKEND) && go build -o ../$(BIN) .
	@echo "built ./$(BIN)"

.PHONY: run
run: build ## Build everything and run the server
	GTMA_STATIC=$(FRONTEND)/dist ./$(BIN) -addr $(ADDR)

.PHONY: test-db
test-db: ## Start a throwaway MariaDB on :13306 (root/secret)
	docker run -d --name gtma-db -e MARIADB_ROOT_PASSWORD=secret -p 13306:3306 mariadb:11
	@echo "MariaDB on 127.0.0.1:13306  user=root  pass=secret"

.PHONY: test-db-stop
test-db-stop: ## Remove the throwaway MariaDB
	docker rm -f gtma-db

.PHONY: tidy
tidy: ## go mod tidy
	cd $(BACKEND) && go mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN) $(FRONTEND)/dist
