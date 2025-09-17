FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    python3 \
    python3-pip \
    openmpi-bin \
    libopenmpi-dev \
    git \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip3 install flask flask-cors numpy matplotlib pandas

# Download and compile SU2
WORKDIR /opt
RUN git clone --depth 1 --branch v7.5.1 https://github.com/su2code/SU2.git su2

WORKDIR /opt/su2
RUN ./meson.py build -Dwith-mpi=enabled -Denable-autodiff=false \
    && ./ninja -C build install

# Set environment variables
ENV SU2_RUN=/opt/su2/bin
ENV PATH="${SU2_RUN}:${PATH}"
ENV PYTHONPATH="/opt/su2/bin:${PYTHONPATH}"

# Create working directory
WORKDIR /workspace

# Copy API server
COPY lib/api_server.py /app/api_server.py

# Expose API port
EXPOSE 9514

# Run API server
CMD ["python3", "/app/api_server.py"]
