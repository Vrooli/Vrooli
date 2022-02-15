FROM node:16-alpine
EXPOSE 3000

# Set variables
ARG PROJECT_DIR
ENV main=packages/ui
ENV shared=packages/shared
ENV scripts=${PROJECT_DIR}/scripts

# Create directories with correct permissions
RUN mkdir -p ${PROJECT_DIR}/${main}/node_modules/.cache ${PROJECT_DIR}/${shared}/node_modules/.cache &&\
    mkdir -p ${PROJECT_DIR}/${main}/build &&\
    chown -R node:node ${PROJECT_DIR}

# Install global packages (must be done as the root user)
RUN yarn global add react-scripts serve

# Switch to a user with less permissions
USER node

# Set working directory
WORKDIR ${PROJECT_DIR}

# Copy packages over first. This helps with caching
COPY --chown=node:node package.json package.json
COPY --chown=node:node ${shared}/package.json ${shared}/tsconfig.json ${shared}/
COPY --chown=node:node ${main}/package.json ${main}/tsconfig.json ${main}/

# Install packages
RUN yarn install

# Copy rest of repo over
COPY --chown=node:node ${shared}/src ${shared}/
COPY --chown=node:node ${main}/src ${main}/public ${main}/
COPY --chown=node:node scripts/ui.sh scripts/wait-for.sh ${scripts}/

USER root
