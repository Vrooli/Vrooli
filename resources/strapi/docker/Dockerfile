# Strapi v5 Docker Image
FROM node:20-alpine

# Install build dependencies
RUN apk add --no-cache build-base gcc autoconf automake zlib-dev libpng-dev vips-dev git

# Set working directory
WORKDIR /app

# Install PM2 and Strapi CLI globally
RUN npm install -g pm2 @strapi/strapi@^5

# Copy package file first for better caching
COPY package.json ./

# Install dependencies
RUN npm install

# Copy initialization scripts
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Create necessary directories
RUN mkdir -p /app/public /app/config /app/src/api

# Expose port
EXPOSE 1337

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=3 \
  CMD node -e "require('http').get('http://localhost:1337/_health', (r) => {process.exit(r.statusCode === 204 ? 0 : 1)})" || exit 1

# Start Strapi using entrypoint
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["npm", "run", "start"]