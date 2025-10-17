FROM ultralytics/ultralytics:latest

WORKDIR /app

# Install additional dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy API server
COPY api_server.py .

# Create model directory
RUN mkdir -p /models

EXPOSE 11455

CMD ["python", "api_server.py"]
