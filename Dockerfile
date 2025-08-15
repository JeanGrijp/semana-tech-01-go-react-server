# Dockerfile para API Go + React Server
# Multi-stage build para otimizar tamanho da imagem

# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates tzdata

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Download das dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Build da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/wsrs/main.go

# Stage 2: Final image
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Criar usuário não-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Criar diretórios necessários
RUN mkdir -p /app/internal/logger/logs && \
    chown -R appuser:appgroup /app

WORKDIR /app

# Copiar binário do stage de build
COPY --from=builder /app/main .
COPY --from=builder /app/.env.example .env

# Mudar ownership para usuário não-root
RUN chown appuser:appgroup main

# Trocar para usuário não-root
USER appuser

# Expor porta
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/rooms || exit 1

# Comando para executar a aplicação
CMD ["./main"]
