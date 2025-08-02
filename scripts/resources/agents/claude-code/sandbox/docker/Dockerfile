# Claude Code Sandbox Container
# Provides isolated environment for testing claude-code capabilities
FROM node:18-alpine

# Install required dependencies
RUN apk add --no-cache \
    git \
    bash \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Install Claude Code globally
RUN npm install -g @anthropic-ai/claude-code@latest

# Create non-root user for security
RUN addgroup -g 1001 -S claude && \
    adduser -S claude -u 1001 -G claude -h /home/claude -s /bin/bash

# Create necessary directories with proper ownership
RUN mkdir -p /home/claude/.claude \
             /home/claude/.npm \
             /workspace \
             /tmp/claude-sandbox && \
    chown -R claude:claude /home/claude /workspace /tmp/claude-sandbox

# Set restrictive permissions
RUN chmod 700 /home/claude/.claude

# Switch to non-root user
USER claude
WORKDIR /workspace

# Set environment variables for safer operation
ENV NODE_ENV=production \
    CLAUDE_CONFIG_DIR=/home/claude/.claude \
    CLAUDE_SANDBOX=true \
    HOME=/home/claude

# Add security labels
LABEL security.sandbox="true" \
      security.user="non-root" \
      security.purpose="claude-code-testing" \
      maintainer="Vrooli Resource Management"

# Health check to ensure claude is available
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD claude --version || exit 1

# Default entrypoint with safety message
ENTRYPOINT ["/bin/bash", "-c", "echo '🔒 Claude Code Sandbox Environment' && echo '⚠️  This is an isolated container for testing' && echo '' && exec claude \"$@\"", "--"]