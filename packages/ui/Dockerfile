# Stage 0. Copy required files
FROM node:22-alpine AS stage0

# Set working directory
ARG PROJECT_DIR
ARG VIRTUAL_PORT
WORKDIR ${PROJECT_DIR}

# Copy required files
COPY --chown=node:node package.json pnpm-lock.yaml ./
COPY --chown=node:node packages/ui packages/ui/
COPY --chown=node:node packages/shared packages/shared/
COPY --chown=node:node scripts/* scripts/

# Assign working directory to node
RUN chown -R node:node .

# Stage 1. Copy files from stage 0, and install pnpm packages
FROM node:22-alpine AS stage1

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 0
COPY --from=stage0 ${PROJECT_DIR} ./

# Install project dependencies
RUN pnpm install --network-timeout 600000

# Install global packages
RUN pnpm global add vite@5.4.7 --network-timeout 600000

# Stage 2. Copy files from stage 1
FROM node:22-alpine AS stage2

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 1
COPY --from=stage1 ${PROJECT_DIR} ./

# Copy global pnpm packages from stage 1
COPY --from=stage1 /usr/local/share/.config/pnpm/global /usr/local/share/.config/pnpm/global

# Set port
EXPOSE ${VIRTUAL_PORT}
