FROM python:3.11-slim

WORKDIR /app

# Install system dependencies including Java for Spark
RUN apt-get update && apt-get install -y \
    curl \
    gcc \
    g++ \
    default-jre-headless \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages including PySpark
RUN pip install --no-cache-dir \
    splink==3.9.14 \
    duckdb==0.10.0 \
    fastapi==0.109.0 \
    uvicorn==0.27.0 \
    pydantic==2.5.0 \
    pandas==2.1.4 \
    pyarrow==14.0.2 \
    plotly==5.18.0 \
    redis[hiredis]==5.0.1 \
    psycopg2-binary==2.9.9 \
    boto3==1.34.0 \
    pyspark==3.5.0

# Copy application code
COPY api/ /app/api/
COPY lib/ /app/lib/

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV SPLINK_PORT=8096
ENV SPLINK_BACKEND=duckdb

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${SPLINK_PORT}/health || exit 1

# Start the service
CMD ["python", "-m", "uvicorn", "api.main:app", "--host", "0.0.0.0", "--port", "8096"]
