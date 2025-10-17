# Multi-stage build for Code Smell Detector

# Stage 1: Build Go API
FROM golang:1.21-alpine AS go-builder
WORKDIR /build
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN go build -o code-smell-api main.go

# Stage 2: Build Node.js components
FROM node:18-alpine AS node-builder
WORKDIR /build
COPY package*.json ./
RUN npm ci --only=production
COPY lib/ ./lib/
COPY cli/ ./cli/

# Stage 3: Final image
FROM node:18-alpine
RUN apk add --no-cache bash

WORKDIR /app

# Copy Go binary
COPY --from=go-builder /build/code-smell-api /app/api/

# Copy Node.js dependencies and code
COPY --from=node-builder /build/node_modules /app/node_modules
COPY --from=node-builder /build/lib /app/lib
COPY --from=node-builder /build/cli /app/cli

# Copy configuration and rules
COPY .vrooli/ /app/.vrooli/
COPY initialization/ /app/initialization/
COPY ui/ /app/ui/
COPY package.json /app/

# Set environment variables
ENV CODE_SMELL_API_PORT=8090
ENV CODE_SMELL_UI_PORT=3090
ENV NODE_ENV=production

# Create startup script
RUN echo '#!/bin/sh' > /app/start.sh && \
    echo 'cd /app && ./api/code-smell-api &' >> /app/start.sh && \
    echo 'cd /app/ui && python3 -m http.server 3090 &' >> /app/start.sh && \
    echo 'wait' >> /app/start.sh && \
    chmod +x /app/start.sh

EXPOSE 8090 3090

CMD ["/app/start.sh"]