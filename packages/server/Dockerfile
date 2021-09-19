FROM node:14.17-alpine
EXPOSE 5000

# Set variables
ARG PROJECT_DIR
ENV server=packages/server
ENV shared=packages/shared
ENV scripts=${PROJECT_DIR}/scripts

# Create directories with correct permissions
RUN mkdir -p ${PROJECT_DIR}/assets/images ${PROJECT_DIR}/assets/public ${PROJECT_DIR}/assets/private &&\ 
    mkdir -p ${PROJECT_DIR}/${ui}/node_modules/.cache ${PROJECT_DIR}/${server}/node_modules/.cache ${PROJECT_DIR}/${shared}/node_modules/.cache &&\
    mkdir -p ${PROJECT_DIR}/${ui}/build &&\
    chown -R node:node ${PROJECT_DIR}

# Install global packages (must be done as the root user)
RUN yarn global add knex@0.95.5 nodemon prisma

# Switch to a user with less permissions
USER node

# Set working directory
WORKDIR ${PROJECT_DIR}

# Copy packages over first. This helps with caching
COPY --chown=node:node package.json package.json
COPY --chown=node:node ${shared}/package.json ${shared}/package.json
COPY --chown=node:node ${server}/package.json ${server}/package.json

# Install packages
RUN yarn install

# Copy rest of repo over
COPY --chown=node:node ${shared}/index.js ${shared}/src ${shared}/
COPY --chown=node:node ${server}/src ${server}/tools ${server}/
COPY --chown=node:node scripts/server.sh scripts/wait-for.sh ${scripts}/

USER root
