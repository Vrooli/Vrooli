FROM nodered/node-red:latest-18

USER root

# Install additional packages for Vrooli integration
RUN apk add --no-cache \
    bash \
    curl \
    jq \
    python3 \
    py3-pip \
    git \
    docker-cli \
    gcc \
    g++ \
    make \
    linux-headers \
    coreutils \
    findutils \
    grep \
    sed \
    gawk \
    tar \
    gzip \
    bzip2 \
    xz \
    wget \
    util-linux \
    shadow

# Create directories for host access
RUN mkdir -p /host/bin /host/usr/bin /workspace /data/flows /data/nodes

# Install useful Node-RED nodes
USER node-red
RUN npm install --no-audit --no-update-notifier --no-fund --save \
    node-red-dashboard \
    node-red-contrib-fs \
    node-red-contrib-dockerode

# Copy custom entrypoint
USER root
COPY docker-entrypoint.sh /usr/src/node-red/docker-entrypoint.sh
RUN chmod +x /usr/src/node-red/docker-entrypoint.sh

# Set working directory
WORKDIR /usr/src/node-red

# Use custom entrypoint (runs as root, then drops to node-red user)
ENTRYPOINT ["/usr/src/node-red/docker-entrypoint.sh"]

# Default command to start Node-RED
CMD []

# Expose port
EXPOSE 1880

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:1880/ || exit 1