# Stage 0. Copy required files
FROM node:18-alpine3.20 AS stage0 

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy main package.json, as well as packages this container will use. Doing this first helps with caching (I think).
COPY --chown=node:node package.json pnpm-lock.yaml ./
COPY --chown=node:node packages/jobs/package.json packages/jobs/
COPY --chown=node:node packages/server/package.json packages/server/
COPY --chown=node:node packages/shared/package.json packages/shared/

# Copy all files from the package.json paths above
COPY --chown=node:node packages/jobs/src packages/jobs/
COPY --chown=node:node packages/server/src packages/server/
COPY --chown=node:node packages/shared/src packages/shared/

# Copy required script files
COPY --chown=node:node scripts/* scripts/

# Assign working directory to node
RUN chown -R node:node .

# Stage 1. Copy files from stage 0, and install pnpm packages
FROM node:18-alpine3.20 AS stage1

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 2, and install pnpm packages
COPY --from=stage0 ${PROJECT_DIR} ./

# Install local packages
RUN pnpm install --frozen-lockfile --network-timeout 600000

# Install global packages
RUN pnpm global add typescript@5.3.3 nodemon@3.0.2 prisma@6.1.0 --network-timeout 600000

# Stage 2. Copy files from stage 1, and install required unix tools
FROM node:18-alpine3.20 AS stage2

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 3
COPY --from=stage1 ${PROJECT_DIR} ./

# Copy global pnpm packages from stage 3
COPY --from=stage1 /usr/local/share/.config/pnpm/global /usr/local/share/.config/pnpm/global

# Update and install unix packages
RUN apk update && \
    apk add --no-cache netcat-openbsd

# Set debugger port
EXPOSE 9230
