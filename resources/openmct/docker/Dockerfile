FROM node:18-alpine

# Install dependencies
RUN apk add --no-cache git python3 make g++ sqlite

# Create app directory
WORKDIR /app

# Clone Open MCT
RUN git clone https://github.com/nasa/openmct.git . && \
    git checkout v2.2.2

# Install npm dependencies
RUN npm install

# Copy custom configuration
COPY server.js /app/server.js
COPY telemetry-server.js /app/telemetry-server.js

# Create data directory
RUN mkdir -p /data

# Expose ports
EXPOSE 8099 8100

# Start server
CMD ["node", "server.js"]
