FROM ubuntu:22.04

# Avoid interactive prompts during build
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    gfortran \
    git \
    libopenmpi-dev \
    openmpi-bin \
    liblapack-dev \
    libblas-dev \
    libmumps-dev \
    libhypre-dev \
    libqt5opengl5-dev \
    qtscript5-dev \
    libqwt-qt5-dev \
    python3 \
    python3-pip \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies for API server
RUN pip3 install flask requests numpy pandas matplotlib

# Clone and build Elmer from source (most reliable approach)
WORKDIR /opt
RUN git clone --depth 1 --branch release https://github.com/ElmerCSC/elmerfem.git && \
    cd elmerfem && \
    mkdir build && cd build && \
    cmake .. \
        -DCMAKE_INSTALL_PREFIX=/usr/local \
        -DWITH_MPI=TRUE \
        -DWITH_OpenMP=TRUE \
        -DWITH_MKL=FALSE \
        -DWITH_ELMERGUI=FALSE \
        -DWITH_MATC=TRUE \
        -DWITH_LUA=FALSE && \
    make -j$(nproc) && \
    make install && \
    cd / && rm -rf /opt/elmerfem

# Set environment variables for Elmer
ENV ELMER_HOME=/usr/local
ENV PATH="${ELMER_HOME}/bin:${PATH}"
ENV LD_LIBRARY_PATH="${ELMER_HOME}/lib:${LD_LIBRARY_PATH}"

# Create workspace structure
WORKDIR /workspace
RUN mkdir -p /workspace/cases /workspace/meshes /workspace/results /workspace/examples

# Copy API server and examples
COPY lib/api_server.py /workspace/
COPY examples/ /workspace/examples/

# Expose API port (from port_registry.sh)
EXPOSE 8192

# Default command to run API server
CMD ["python3", "/workspace/api_server.py"]