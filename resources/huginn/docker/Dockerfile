FROM ghcr.io/huginn/huginn-single-process:latest

USER root

# Install additional packages for Vrooli integration
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    python3 \
    python3-pip \
    git \
    docker.io \
    netcat-openbsd \
    mysql-client \
    redis-tools \
    vim \
    less \
    htop \
    && rm -rf /var/lib/apt/lists/*

# Create directories for custom configurations
RUN mkdir -p /workspace /app/uploads

# Skip gem additions for now - the base image has requirements

# Copy custom entrypoint
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Set working directory
WORKDIR /app

# Use custom entrypoint
ENTRYPOINT ["/docker-entrypoint.sh"]

# Default command
CMD ["/scripts/init"]

# Expose port
EXPOSE 3000

# Health check for Huginn
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:3000/users/sign_in || exit 1