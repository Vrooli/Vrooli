FROM ubuntu:22.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    python3 \
    python3-pip \
    python3-dev \
    python3-numpy \
    python3-scipy \
    python3-matplotlib \
    libhdf5-dev \
    libhdf5-openmpi-dev \
    libfftw3-dev \
    libgsl-dev \
    liblapack-dev \
    libblas-dev \
    openmpi-bin \
    libopenmpi-dev \
    wget \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install MEEP via conda (most reliable method)
RUN wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh && \
    bash Miniconda3-latest-Linux-x86_64.sh -b -p /opt/conda && \
    rm Miniconda3-latest-Linux-x86_64.sh

ENV PATH="/opt/conda/bin:${PATH}"

# Configure conda to use conda-forge only (avoiding license issues)
RUN conda config --add channels conda-forge && \
    conda config --set channel_priority strict && \
    conda config --remove channels defaults || true

# Install MEEP with MPI support using override channels to avoid ToS issues
RUN conda install -y --override-channels -c conda-forge pymeep=*=mpi_mpich_* h5py matplotlib flask && \
    conda clean -a -y

# Install additional Python packages
RUN pip3 install --no-cache-dir \
    fastapi \
    uvicorn \
    pydantic \
    aiofiles \
    psycopg2-binary \
    minio

# Create working directory
WORKDIR /app

# Copy API server
COPY lib/api_server.py /app/api_server.py
COPY examples /app/templates

# Create data directories
RUN mkdir -p /data/results /data/templates

# Expose API port
EXPOSE 8193

# Run API server
CMD ["python3", "/app/api_server.py"]