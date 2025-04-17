# Stage 0: Base Node environment
FROM node:18-alpine3.20 AS base
ARG PROJECT_DIR=/srv/app
WORKDIR ${PROJECT_DIR}

# Stage 1: Install all dependencies
FROM base AS deps
# Copy root package config
COPY package.json yarn.lock ./
# Copy workspace package configs that server depends on
COPY packages/server/package.json ./packages/server/
COPY packages/shared/package.json ./packages/shared/
# Add other workspace package.json files if server depends on them
# Copy shared package source code for workspace resolution
COPY packages/shared ./packages/shared/

# Install all dependencies (including devDependencies needed for build)
RUN yarn install --frozen-lockfile --network-timeout 600000

# Stage 2: Build the application
FROM base AS builder
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}
# Copy source code
COPY . .
# Copy installed node_modules from deps stage
COPY --from=deps ${PROJECT_DIR}/node_modules ./node_modules

# Build shared package (assuming yarn build is the command)
# Adjust if shared package build command is different
RUN yarn workspace @local/shared run build

# Build server package (assuming yarn build is the command)
# This should also trigger prisma generate if configured in package.json postinstall/build
RUN yarn workspace @local/server run build

# --- Optional: Prune dev dependencies ---
# If the final image size is critical, we can prune dev dependencies here.
# RUN yarn workspaces focus --all --production

# Stage 3: Final production image
FROM node:18-alpine3.20 AS final
ENV NODE_ENV=production
ARG PROJECT_DIR=/srv/app
ARG VIRTUAL_PORT=5329 # Default port, can be overridden
WORKDIR ${PROJECT_DIR}

# Install necessary OS packages (Prisma needs openssl)
RUN apk update && \
    apk add --no-cache openssl netcat-openbsd && \
    rm -rf /var/cache/apk/*

# Copy necessary package files for runtime
COPY --chown=node:node package.json yarn.lock ./
COPY --chown=node:node packages/server/package.json ./packages/server/
COPY --chown=node:node packages/shared/package.json ./packages/shared/

# Use node_modules from builder to preserve workspace links
COPY --from=builder --chown=node:node ${PROJECT_DIR}/node_modules ./node_modules

# Copy built application code from the builder stage
COPY --from=builder --chown=node:node ${PROJECT_DIR}/packages/server/dist ./packages/server/dist
COPY --from=builder --chown=node:node ${PROJECT_DIR}/packages/shared/dist ./packages/shared/dist

# Copy runtime scripts
COPY --from=builder --chown=node:node ${PROJECT_DIR}/scripts ./scripts

# Install global dependencies needed for runtime/development
RUN yarn global add typescript@5.4.5 nodemon@3.0.2 prisma@6.1.0 @swc/cli@^0.1.63 @swc/core@^1.3.104 copyfiles@^2.4.1 --network-timeout 600000

# Copy Prisma schema if it's not included in the server dist
# Adjust path if schema.prisma is located elsewhere or included in build output
# COPY --from=builder --chown=node:node ${PROJECT_DIR}/packages/server/prisma ./packages/server/prisma <- REMOVED: Assuming build output in dist includes necessary client files

# Expose port
EXPOSE ${VIRTUAL_PORT} 9229

# Define the command to run the application
# Adjust the path to the server entrypoint if necessary
CMD ["node", "packages/server/dist/index.js"]
