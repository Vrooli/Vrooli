FROM python:3.12-slim

# Install SimPy and common dependencies
RUN pip install --no-cache-dir \
    simpy \
    numpy \
    matplotlib \
    pandas \
    scipy

# Create working directory
WORKDIR /simulations

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV SIMPY_OUTPUT_DIR=/outputs
ENV SIMPY_LOG_DIR=/logs

# Create directories
RUN mkdir -p /simulations /outputs /logs

# Default command
CMD ["python", "--version"]