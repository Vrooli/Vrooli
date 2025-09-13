FROM opencfd/openfoam:v2312

# Install Python and API dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    python3-flask \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create working directories
RUN mkdir -p /cases /data /results

# Set environment
ENV OPENFOAM_VERSION=v2312
ENV WM_PROJECT_DIR=/opt/OpenFOAM/OpenFOAM-v2312

# Copy API server (will be mounted at runtime)
WORKDIR /opt

# Default command
CMD ["tail", "-f", "/dev/null"]