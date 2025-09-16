# PostGIS with pgRouting extension
FROM postgis/postgis:16-3.4

# Install pgRouting and additional spatial tools
RUN apt-get update && apt-get install -y \
    postgresql-16-pgrouting \
    gdal-bin \
    proj-bin \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create initialization script for pgRouting
RUN echo "CREATE EXTENSION IF NOT EXISTS pgrouting;" > /docker-entrypoint-initdb.d/10-pgrouting.sql

# Health check configuration
HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
    CMD pg_isready -U vrooli -d spatial || exit 1

# Labels for documentation
LABEL maintainer="Vrooli Team" \
      description="PostGIS with pgRouting for advanced spatial analysis" \
      version="16-3.4-routing"