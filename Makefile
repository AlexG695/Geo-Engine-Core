# ==============================================================================
# 🌍 Geo Engine Core - Makefile
# ==============================================================================

# Configuration Variables
DB_URL=postgres://geo:geoengine_secret@localhost:5432/geoengine?sslmode=disable
MIGRATION_PATH=backend/sql/schema

.PHONY: help up down logs test test-backend test-frontend sqlc migrate-up migrate-down run-backend run-frontend install

# ==============================================================================
# 🛠️ Main Commands
# ==============================================================================

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start the full stack with Docker (Backend + Frontend + DB + Redis)
	docker-compose up --build -d
	@echo "🚀 Geo Engine is running!"
	@echo "Frontend: http://localhost:5173"
	@echo "Backend:  http://localhost:8080"

down: ## Stop and remove containers
	docker-compose down
	@echo "🛑 Services stopped."

logs: ## Tail container logs in real-time
	docker-compose logs -f

# ==============================================================================
# ⚙️ Backend (Go)
# ==============================================================================

test-backend: ## Run Backend tests (Integration & Concurrency)
	@echo "🧪 Running Backend Tests..."
	cd backend && go test ./... -v

run-backend: ## Run Backend locally (requires DB running)
	cd backend && go run cmd/api/main.go

sqlc: ## Generate Go code from SQL (requires sqlc installed)
	cd backend && sqlc generate
	@echo "✅ SQLC Code Generated"

# Database Migrations (Requires golang-migrate)
migrate-create: ## Create a new migration. Usage: make migrate-create name=filename
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $(name)

migrate-up: ## Apply database migrations
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" up

migrate-down: ## Rollback the last migration
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" down 1

# ==============================================================================
# 💻 Frontend (React)
# ==============================================================================

install-frontend: ## Install Frontend dependencies
	cd frontend && npm install

test-frontend: ## Run Frontend tests (Vitest)
	@echo "🧪 Running Frontend Tests..."
	cd frontend && npx vitest

run-frontend: ## Run Frontend locally
	cd frontend && npm run dev

# ==============================================================================
# ✅ Quality & Utils
# ==============================================================================

test: test-backend test-frontend ## Run ALL tests (Back & Front)
	@echo "🎉 All tests passed!"

clean: ## Clean temporary files and orphan containers
	docker system prune -f
	rm -rf backend/tmp
	rm -rf frontend/dist