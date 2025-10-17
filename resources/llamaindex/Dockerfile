FROM python:3.11-slim

WORKDIR /app

# Install LlamaIndex and dependencies
RUN pip install --no-cache-dir \
    llama-index==0.11.14 \
    llama-index-embeddings-ollama \
    llama-index-embeddings-huggingface \
    llama-index-llms-ollama \
    llama-index-llms-openai \
    llama-index-embeddings-openai \
    fastapi \
    uvicorn \
    python-multipart

# Copy server script (will be mounted as volume)
VOLUME /app/config
VOLUME /app/data

# Set environment variables
ENV LLAMAINDEX_PORT=8090
ENV LLAMAINDEX_DATA_DIR=/app/data

# Run server
CMD ["python", "/app/config/server.py"]