FROM docker.n8n.io/n8nio/n8n:latest

USER root

# Install essential tools that should be container-native
RUN apk add --no-cache \
    bash \
    git \
    openssh-client \
    curl \
    wget \
    jq \
    python3 \
    py3-pip \
    docker-cli

# Create mount points
RUN mkdir -p /host/bin /host/usr/bin /host/usr/local/bin /host/home

# Add custom entrypoint
COPY docker-entrypoint.sh /custom-entrypoint.sh
RUN chmod +x /custom-entrypoint.sh

# Create wrapper scripts for problematic commands
RUN mkdir -p /usr/local/bin

# Tput wrapper (handles terminal capability differences)
RUN printf '#!/bin/sh\nif [ -x /host/usr/bin/tput ]; then\n  TERM=xterm /host/usr/bin/tput "$@"\nelse\n  echo "tput not available" >&2\n  exit 1\nfi\n' > /usr/local/bin/tput && \
    chmod +x /usr/local/bin/tput

USER node

# Use custom entrypoint
ENTRYPOINT ["/custom-entrypoint.sh"]