FROM redis:7-alpine
EXPOSE 6379

# Set variables
ARG PROJECT_DIR

# Create directories for data dumps and appendonly.aof files
RUN mkdir -p ${PROJECT_DIR}/data/redis ${PROJECT_DIR}/data/redis/appendonlydir &&\
    chown -R root:root ${PROJECT_DIR}