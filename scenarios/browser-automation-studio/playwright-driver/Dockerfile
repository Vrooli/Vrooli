# Playwright Driver - Production Dockerfile
# Multi-stage build for minimal final image

# Stage 1: Builder
FROM node:20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache python3 make g++

# Copy package files
COPY package*.json ./
COPY tsconfig*.json ./

# Install dependencies
RUN npm ci --only=production && \
    npm cache clean --force

# Copy source code
COPY src/ ./src/

# Build TypeScript
RUN npm run build

# Stage 2: Runtime
FROM node:20-alpine

# Install Playwright system dependencies
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont

# Set Playwright to use system Chromium
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
    PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium-browser

WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S playwright && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G playwright -g playwright playwright && \
    mkdir -p /app/dist /tmp/playwright && \
    chown -R playwright:playwright /app /tmp/playwright

# Copy built application from builder
COPY --from=builder --chown=playwright:playwright /app/node_modules ./node_modules
COPY --from=builder --chown=playwright:playwright /app/dist ./dist
COPY --from=builder --chown=playwright:playwright /app/package.json ./

# Switch to non-root user
USER playwright

# Expose port
EXPOSE 39400

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:39400/health', (r) => process.exit(r.statusCode === 200 ? 0 : 1))"

# Start server
CMD ["node", "dist/server.js"]
