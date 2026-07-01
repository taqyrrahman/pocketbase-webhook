# syntax=docker/dockerfile:1
FROM golang:alpine AS builder

# Set destination for COPY
WORKDIR /app

# Download Go modules (cached via BuildKit)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source code
COPY . .

# Build with BuildKit cache mounts for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /pocketbase-webhook .

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /app

# ca-certificates for HTTPS webhook calls
RUN apk add --no-cache ca-certificates

# Environment defaults (can override in docker run / compose)
ENV WEBHOOK_URL
ENV WEBHOOK_API_KEY

# Copy the compiled binary from builder stage
COPY --from=builder /pocketbase-webhook .

# PocketBase data directory (important for persistence)
RUN mkdir -p /app/pb_data

# Document the TCP port the application listens on
EXPOSE 8090

# Run pocketbase
CMD ["./pocketbase-webhook", "serve", "--http=0.0.0.0:8090"]