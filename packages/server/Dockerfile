# Stage 0. Copy required files
FROM node:16-alpine as stage0

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy main package.json, as well as packages this container will use. Doing this first helps with caching (I think).
COPY --chown=node:node package.json yarn.lock ./
COPY --chown=node:node packages/server/package.json packages/server/
COPY --chown=node:node packages/shared/consts/package.json packages/shared/consts/
COPY --chown=node:node packages/shared/route/package.json packages/shared/route/
COPY --chown=node:node packages/shared/utils/package.json packages/shared/utils/
COPY --chown=node:node packages/shared/validation/package.json packages/shared/validation/

# Copy all files from packages/server/src, and every directory's src in packages/shared
COPY --chown=node:node packages/server/src packages/server/
COPY --chown=node:node packages/shared/consts/src packages/shared/consts/
COPY --chown=node:node packages/shared/route/src packages/shared/route/
COPY --chown=node:node packages/shared/utils/src packages/shared/utils/
COPY --chown=node:node packages/shared/validation/src packages/shared/validation/

# Copy required script files
COPY --chown=node:node scripts/* scripts/

# Assign working directory to node
RUN chown -R node:node .

# Stage 1. Copy files from stage 0, and install yarn packages
FROM node:16-alpine as stage1

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 2, and install yarn packages
COPY --from=stage0 ${PROJECT_DIR} ./

# Install local packages
RUN yarn install

# Install global packages
RUN yarn global add typescript ts-node nodemon prisma@4.0.0

# Stage 2. Copy files from stage 1, and install required unix tools
FROM node:16-alpine as stage2

# Set working directory
ARG PROJECT_DIR
WORKDIR ${PROJECT_DIR}

# Copy entire working directory contents from stage 3
COPY --from=stage1 ${PROJECT_DIR} ./

# Copy global yarn packages from stage 3
COPY --from=stage1 /usr/local/share/.config/yarn/global /usr/local/share/.config/yarn/global

# Update and install unix packages
RUN apk update
RUN apk add netcat-openbsd

# Set port
EXPOSE 5329