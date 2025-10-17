# Build stage for Go API
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY main.go .

RUN go mod init restic-api && \
    go build -o restic-api main.go

# Final stage with restic and API
FROM restic/restic:latest

# Install runtime dependencies and cron for scheduling
RUN apk add --no-cache ca-certificates dcron curl

# Copy API server
COPY --from=builder /app/restic-api /usr/local/bin/

# Copy scripts
COPY backup-scheduler.sh /usr/local/bin/
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/*.sh

# Set environment variables (password will be set at runtime)
ENV RESTIC_REPOSITORY=/repository
ENV RESTIC_CACHE_DIR=/cache
ENV API_PORT=8000
ENV ENABLE_SCHEDULER=true
ENV BACKUP_SCHEDULE="0 2 * * *"

# Create necessary directories
RUN mkdir -p /repository /cache /config /data

# Expose API port
EXPOSE 8000

# Run the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]