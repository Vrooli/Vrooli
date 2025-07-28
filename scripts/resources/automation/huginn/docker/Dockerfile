FROM ghcr.io/huginn/huginn:latest

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
    postgresql-client \
    redis-tools \
    vim \
    less \
    htop \
    && rm -rf /var/lib/apt/lists/*

# Create directories for host access and custom configurations
RUN mkdir -p /host/bin /host/usr/bin /workspace /app/uploads

# Install additional Ruby gems for enhanced functionality
USER huginn
RUN bundle add rest-client json colorize --skip-install && \
    bundle install

# Copy custom entrypoint
USER root
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