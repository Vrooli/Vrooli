# Stage 0. Copy required files
FROM node:16-alpine as stage0

# Set working directory
ARG PROJECT_DIR
ARG VIRTUAL_PORT
WORKDIR ${PROJECT_DIR}

# Copy required files
COPY --chown=node:node package.json yarn.lock ./
COPY --chown=node:node packages/ui packages/ui/
COPY --chown=node:node packages/shared packages/shared/
COPY --chown=node:node scripts/* scripts/

# Assign working directory to node
RUN chown -R node:node .

# Set port
EXPOSE ${VIRTUAL_PORT}