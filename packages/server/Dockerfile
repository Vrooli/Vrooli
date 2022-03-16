FROM node:16-alpine
EXPOSE 5329

# Set variables
ARG PROJECT_DIR
ENV server=packages/server
ENV shared=packages/shared
ENV scripts=${PROJECT_DIR}/scripts

# Create directories with correct permissions
RUN mkdir -p ${PROJECT_DIR}/${server}/node_modules/.cache ${PROJECT_DIR}/${shared}/node_modules/.cache &&\
    chown -R node:node ${PROJECT_DIR}

# Install global packages (must be done as the root user)
RUN yarn global add typescript ts-node nodemon prisma@3.10.0

# Switch to a user with less permissions
USER node

# Set working directory
WORKDIR ${PROJECT_DIR}

# Copy packages over first. This helps with caching
COPY --chown=node:node package.json package.json
COPY --chown=node:node ${shared}/package.json ${shared}/tsconfig.json ${shared}/
COPY --chown=node:node ${server}/package.json ${server}/tsconfig.json ${server}/nodemon.json ${server}/

# Install packages
RUN yarn install

# Copy rest of repo over
COPY --chown=node:node ${shared}/src ${shared}/
COPY --chown=node:node ${server}/src ${server}/
COPY --chown=node:node scripts/server.sh scripts/wait-for.sh ${scripts}/

USER root

# Update and install unix packages
RUN apk update
RUN apk add netcat-openbsd