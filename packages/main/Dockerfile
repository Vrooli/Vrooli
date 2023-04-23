# Stage 0. Copy required files
# NOTE: Using alpine3.16 because of this issue: https://github.com/prisma/prisma/issues/16553
FROM node:16-alpine3.16 as stage0 

# Set working directory
ARG PROJECT_DIR
ARG VIRTUAL_PORT
WORKDIR ${PROJECT_DIR}

# Copy all package.json files. Doing this first helps with caching (I think).
COPY --chown=node:node package.json yarn.lock ./
COPY --chown=node:node packages/main/package.json packages/main/
COPY --chown=node:node packages/main/consts/package.json packages/main/consts/
COPY --chown=node:node packages/main/icons/package.json packages/main/icons/
COPY --chown=node:node packages/main/server/package.json packages/main/server/
COPY --chown=node:node packages/main/translations/package.json packages/main/translations/
COPY --chown=node:node packages/main/ui/package.json packages/main/ui/
COPY --chown=node:node packages/main/utils/package.json packages/main/utils/
COPY --chown=node:node packages/main/uuid/package.json packages/main/uuid/
COPY --chown=node:node packages/main/validation/package.json packages/main/validation/

# Copy source files
COPY --chown=node:node packages/main/consts/src packages/main/consts/src/
COPY --chown=node:node packages/main/icons/src packages/main/icons/src/
COPY --chown=node:node packages/main/server/src packages/main/server/src/
COPY --chown=node:node packages/main/translations/src packages/main/translations/src/
COPY --chown=node:node packages/main/ui/src packages/main/ui/src/
COPY --chown=node:node packages/main/utils/src packages/main/utils/src/
COPY --chown=node:node packages/main/uuid/src packages/main/uuid/src/
COPY --chown=node:node packages/main/validation/src packages/main/validation/src/

# Copy other necessary files
COPY --chown=node:node packages/main/ui/index.html packages/main/ui/index.html
COPY --chown=node:node packages/main/ui/public packages/main/ui/public
COPY --chown=node:node packages/main/server/vite.config.ts packages/main/server/vite.config.ts
COPY --chown=node:node packages/main/ui/vite.config.ts packages/main/ui/vite.config.ts

# Copy required script files
COPY --chown=node:node scripts/* scripts/

# Copy RSA public and private key files
COPY --chown=node:node jwt_pub.pem ./
COPY --chown=node:node jwt_priv.pem ./

# Assign working directory to node
RUN chown -R node:node .

# Stage 1. Copy files from stage 0, and install yarn packages
FROM node:16-alpine3.16 as stage1

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 2, and install yarn packages
COPY --from=stage0 ${PROJECT_DIR} ./

# Install local packages
RUN yarn install

# Install global packages
RUN yarn global add typescript ts-node nodemon prisma@4.11.0

# Stage 2. Copy files from stage 1, and install required unix tools
FROM node:16-alpine3.16 as stage2

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 3
COPY --from=stage1 ${PROJECT_DIR} ./

# Copy global yarn packages from stage 3
COPY --from=stage1 /usr/local/share/.config/yarn/global /usr/local/share/.config/yarn/global

# Update and install unix packages
RUN apk update && \
    apk add --no-cache netcat-openbsd

# Set port
EXPOSE ${VIRTUAL_PORT} 9229
