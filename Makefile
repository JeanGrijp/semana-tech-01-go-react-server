# Makefile para API Go + React Server
# ====================================

# Configurações
APP_NAME = wsrs-server
BINARY_DIR = bin
BINARY_PATH = $(BINARY_DIR)/$(APP_NAME)
MAIN_PATH = ./cmd/wsrs/main.go
MIGRATIONS_PATH = ./internal/store/pgstore/migrations

# Variáveis de ambiente padrão para desenvolvimento
export LOG_LEVEL ?= info
export WSRS_DATABASE_HOST ?= localhost
export WSRS_DATABASE_PORT ?= 5432
export WSRS_DATABASE_USER ?= postgres
export WSRS_DATABASE_PASSWORD ?= password
export WSRS_DATABASE_NAME ?= wsrs_db

# Cores para output
GREEN = \033[32m
YELLOW = \033[33m
RED = \033[31m
BLUE = \033[34m
NC = \033[0m # No Color

.PHONY: help
help: ## Mostra este menu de ajuda
	@echo "$(BLUE)API Go + React Server - Comandos Disponíveis:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

# ===========================
# 🏗️  BUILD & DEVELOPMENT
# ===========================

.PHONY: build
build: ## Compila a aplicação
	@echo "$(YELLOW)📦 Compilando aplicação...$(NC)"
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)✅ Aplicação compilada em $(BINARY_PATH)$(NC)"

.PHONY: build-linux
build-linux: ## Compila para Linux (útil para Docker/deploy)
	@echo "$(YELLOW)📦 Compilando para Linux...$(NC)"
	@mkdir -p $(BINARY_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BINARY_PATH)-linux $(MAIN_PATH)
	@echo "$(GREEN)✅ Aplicação compilada para Linux em $(BINARY_PATH)-linux$(NC)"

.PHONY: clean
clean: ## Remove binários e arquivos temporários
	@echo "$(YELLOW)🧹 Limpando arquivos...$(NC)"
	@rm -rf $(BINARY_DIR)
	@rm -rf internal/logger/logs/*.log*
	@rm -rf tmp/
	@go clean
	@echo "$(GREEN)✅ Limpeza concluída$(NC)"

.PHONY: deps
deps: ## Instala/atualiza dependências
	@echo "$(YELLOW)📥 Instalando dependências...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependências atualizadas$(NC)"

.PHONY: deps-upgrade
deps-upgrade: ## Atualiza todas as dependências para versões mais recentes
	@echo "$(YELLOW)⬆️  Atualizando dependências...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✅ Dependências atualizadas$(NC)"

# ===========================
# 🚀 RUN & DEVELOPMENT
# ===========================

.PHONY: run
run: ## Executa a aplicação
	@echo "$(YELLOW)🚀 Iniciando servidor...$(NC)"
	@go run $(MAIN_PATH)

.PHONY: run-debug
run-debug: ## Executa em modo debug
	@echo "$(YELLOW)🐛 Iniciando servidor em modo debug...$(NC)"
	@LOG_LEVEL=debug go run $(MAIN_PATH)

.PHONY: run-bin
run-bin: build ## Compila e executa o binário
	@echo "$(YELLOW)🚀 Executando binário...$(NC)"
	@$(BINARY_PATH)

.PHONY: dev
dev: ## Modo desenvolvimento com Docker Compose
	@echo "$(YELLOW)🔥 Iniciando desenvolvimento com Docker Compose...$(NC)"
	@docker-compose up --build

.PHONY: dev-stop
dev-stop: ## Para o ambiente de desenvolvimento
	@echo "$(YELLOW)🛑 Parando ambiente de desenvolvimento...$(NC)"
	@docker-compose down

.PHONY: dev-logs
dev-logs: ## Mostra logs do ambiente de desenvolvimento
	@echo "$(YELLOW)📋 Logs do ambiente de desenvolvimento...$(NC)"
	@docker-compose logs -f

# ===========================
# 🐘 DATABASE
# ===========================

.PHONY: db-up
db-up: ## Inicia containers do banco de dados
	@echo "$(YELLOW)🐘 Iniciando banco de dados...$(NC)"
	@docker compose up -d db
	@echo "$(GREEN)✅ PostgreSQL iniciado na porta $(WSRS_DATABASE_PORT)$(NC)"

.PHONY: db-up-all
db-up-all: ## Inicia banco + pgAdmin
	@echo "$(YELLOW)🐘 Iniciando banco de dados e pgAdmin...$(NC)"
	@docker compose up -d
	@echo "$(GREEN)✅ PostgreSQL: localhost:$(WSRS_DATABASE_PORT)$(NC)"
	@echo "$(GREEN)✅ pgAdmin: http://localhost:8081 (admin@admin.com / password)$(NC)"

.PHONY: db-down
db-down: ## Para containers do banco
	@echo "$(YELLOW)⏹️  Parando banco de dados...$(NC)"
	@docker compose down
	@echo "$(GREEN)✅ Banco de dados parado$(NC)"

.PHONY: db-restart
db-restart: db-down db-up ## Reinicia o banco de dados

.PHONY: db-logs
db-logs: ## Mostra logs do banco
	@docker compose logs -f db

.PHONY: db-shell
db-shell: ## Acessa shell do PostgreSQL
	@echo "$(YELLOW)🐘 Conectando ao PostgreSQL...$(NC)"
	@docker compose exec db psql -U $(WSRS_DATABASE_USER) -d $(WSRS_DATABASE_NAME)

.PHONY: db-reset
db-reset: ## Remove volumes e reinicia banco (⚠️  APAGA TODOS OS DADOS)
	@echo "$(RED)⚠️  ATENÇÃO: Isso apagará todos os dados do banco!$(NC)"
	@read -p "Tem certeza? [y/N]: " confirm && [ "$$confirm" = "y" ]
	@docker compose down -v
	@docker compose up -d db
	@echo "$(GREEN)✅ Banco de dados resetado$(NC)"

# ===========================
# 🔄 DATABASE MIGRATIONS
# ===========================

.PHONY: migrate-up
migrate-up: ## Executa migrações para cima
	@echo "$(YELLOW)⬆️  Executando migrações...$(NC)"
	@cd $(MIGRATIONS_PATH) && tern migrate
	@echo "$(GREEN)✅ Migrações executadas$(NC)"

.PHONY: migrate-down
migrate-down: ## Reverte última migração
	@echo "$(YELLOW)⬇️  Revertendo migração...$(NC)"
	@cd $(MIGRATIONS_PATH) && tern migrate --destination -1
	@echo "$(GREEN)✅ Migração revertida$(NC)"

.PHONY: migrate-status
migrate-status: ## Mostra status das migrações
	@echo "$(YELLOW)📊 Status das migrações:$(NC)"
	@cd $(MIGRATIONS_PATH) && tern status

.PHONY: migrate-new
migrate-new: ## Cria nova migração (uso: make migrate-new NAME=create_users_table)
	@if [ -z "$(NAME)" ]; then \
		echo "$(RED)❌ Uso: make migrate-new NAME=nome_da_migracao$(NC)"; \
		exit 1; \
	fi
	@cd $(MIGRATIONS_PATH) && tern new $(NAME)
	@echo "$(GREEN)✅ Nova migração criada: $(NAME)$(NC)"

# ===========================
# 🏗️  CODE GENERATION
# ===========================

.PHONY: generate
generate: ## Executa go generate (migrations + sqlc)
	@echo "$(YELLOW)🔧 Executando geradores...$(NC)"
	@go generate ./...
	@echo "$(GREEN)✅ Código gerado$(NC)"

.PHONY: sqlc-generate
sqlc-generate: ## Gera código SQLC apenas
	@echo "$(YELLOW)🔧 Gerando código SQLC...$(NC)"
	@sqlc generate -f ./internal/store/pgstore/sqlc.yaml
	@echo "$(GREEN)✅ Código SQLC gerado$(NC)"

# ===========================
# 🧪 TESTING
# ===========================

.PHONY: test
test: ## Executa todos os testes
	@echo "$(YELLOW)🧪 Executando testes...$(NC)"
	@go test ./...

.PHONY: test-verbose
test-verbose: ## Executa testes com output detalhado
	@echo "$(YELLOW)🧪 Executando testes (verbose)...$(NC)"
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Executa testes com coverage
	@echo "$(YELLOW)🧪 Executando testes com coverage...$(NC)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage gerado em coverage.html$(NC)"

.PHONY: test-race
test-race: ## Executa testes com detecção de race conditions
	@echo "$(YELLOW)🧪 Executando testes (race detection)...$(NC)"
	@go test -race ./...

.PHONY: benchmark
benchmark: ## Executa benchmarks
	@echo "$(YELLOW)📊 Executando benchmarks...$(NC)"
	@go test -bench=. ./...

# ===========================
# 📝 LINTING & FORMATTING
# ===========================

.PHONY: fmt
fmt: ## Formata código Go
	@echo "$(YELLOW)✨ Formatando código...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Código formatado$(NC)"

.PHONY: lint
lint: ## Executa linting com golangci-lint
	@echo "$(YELLOW)🔍 Executando linting...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(RED)❌ golangci-lint não encontrado$(NC)"; \
		echo "$(YELLOW)💡 Instale com: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2$(NC)"; \
	fi

.PHONY: lint-fix
lint-fix: ## Executa linting e corrige problemas automaticamente
	@echo "$(YELLOW)🔧 Executando linting com correções...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run --fix; \
	else \
		echo "$(RED)❌ golangci-lint não encontrado$(NC)"; \
	fi

.PHONY: vet
vet: ## Executa go vet
	@echo "$(YELLOW)🔍 Executando go vet...$(NC)"
	@go vet ./...

.PHONY: check
check: fmt vet lint test ## Executa todas as verificações (fmt + vet + lint + test)

# ===========================
# 📊 MONITORING & LOGS
# ===========================

.PHONY: logs
logs: ## Mostra logs da aplicação
	@echo "$(YELLOW)📋 Logs da aplicação:$(NC)"
	@if [ -f "internal/logger/logs/api.log" ]; then \
		tail -f internal/logger/logs/api.log; \
	else \
		echo "$(RED)❌ Arquivo de log não encontrado$(NC)"; \
	fi

.PHONY: logs-errors
logs-errors: ## Mostra apenas logs de erro
	@echo "$(YELLOW)🚨 Logs de erro:$(NC)"
	@if [ -f "internal/logger/logs/api.log" ]; then \
		grep -i "error\|fatal" internal/logger/logs/api.log | tail -20; \
	else \
		echo "$(RED)❌ Arquivo de log não encontrado$(NC)"; \
	fi

.PHONY: logs-clean
logs-clean: ## Limpa arquivos de log
	@echo "$(YELLOW)🧹 Limpando logs...$(NC)"
	@rm -f internal/logger/logs/*.log*
	@echo "$(GREEN)✅ Logs limpos$(NC)"

# ===========================
# 🚀 DEPLOYMENT
# ===========================

.PHONY: docker-build
docker-build: ## Constrói imagem Docker (requer Dockerfile)
	@echo "$(YELLOW)🐳 Construindo imagem Docker...$(NC)"
	@docker build -t $(APP_NAME):latest .
	@echo "$(GREEN)✅ Imagem Docker criada: $(APP_NAME):latest$(NC)"

.PHONY: docker-run
docker-run: ## Executa aplicação no Docker
	@echo "$(YELLOW)🐳 Executando no Docker...$(NC)"
	@docker run -p 8080:8080 --env-file .env $(APP_NAME):latest

# ===========================
# 🔧 UTILITIES
# ===========================

.PHONY: env-copy
env-copy: ## Copia .env.example para .env
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN)✅ Arquivo .env criado a partir do .env.example$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  Arquivo .env já existe$(NC)"; \
	fi

.PHONY: env-show
env-show: ## Mostra variáveis de ambiente
	@echo "$(BLUE)📋 Variáveis de ambiente atuais:$(NC)"
	@echo "LOG_LEVEL=$(LOG_LEVEL)"
	@echo "WSRS_DATABASE_HOST=$(WSRS_DATABASE_HOST)"
	@echo "WSRS_DATABASE_PORT=$(WSRS_DATABASE_PORT)"
	@echo "WSRS_DATABASE_USER=$(WSRS_DATABASE_USER)"
	@echo "WSRS_DATABASE_NAME=$(WSRS_DATABASE_NAME)"

.PHONY: install-tools
install-tools: ## Instala ferramentas de desenvolvimento
	@echo "$(YELLOW)🔧 Instalando ferramentas de desenvolvimento...$(NC)"
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/jackc/tern/v2@latest
	@echo "$(GREEN)✅ Ferramentas instaladas:$(NC)"
	@echo "  - sqlc (geração de código SQL)"
	@echo "  - tern (migrações)"

.PHONY: size
size: build ## Mostra tamanho do binário
	@echo "$(BLUE)📏 Tamanho do binário:$(NC)"
	@ls -lh $(BINARY_PATH) | awk '{print $$5 " " $$9}'

.PHONY: info
info: ## Mostra informações do projeto
	@echo "$(BLUE)ℹ️  Informações do Projeto:$(NC)"
	@echo "Nome: $(APP_NAME)"
	@echo "Go Version: $$(go version)"
	@echo "Binário: $(BINARY_PATH)"
	@echo "Main: $(MAIN_PATH)"
	@echo "Migrações: $(MIGRATIONS_PATH)"
	@echo ""
	@make env-show

# ===========================
# 🎯 SHORTCUTS ÚTEIS
# ===========================

.PHONY: setup
setup: deps env-copy db-up migrate-up install-tools ## Setup completo do projeto
	@echo "$(GREEN)🎉 Setup completo! Execute 'make run' para iniciar$(NC)"

.PHONY: restart
restart: db-restart run ## Reinicia banco e aplicação

.PHONY: reset
reset: clean db-reset setup ## Reset completo do projeto (⚠️  APAGA DADOS)

# Comandos padrão
.DEFAULT_GOAL := help
