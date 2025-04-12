#!/bin/bash

cd /root/Vrooli

# Pull latest changes if this is a git repository
if [ -d .git ]; then
  git pull
fi

# Check if environment was specified
ENVIRONMENT=${1:-production}

# Ensure environment file exists
if [ "$ENVIRONMENT" == "production" ]; then
  ENV_FILE=".env-prod"
else
  ENV_FILE=".env-dev"
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: $ENV_FILE file not found!"
  exit 1
fi

# Copy env file
cp "$ENV_FILE" .env

# Select the appropriate docker-compose file
if [ "$ENVIRONMENT" == "production" ]; then
  COMPOSE_FILE="docker-compose-prod.yml"
else
  COMPOSE_FILE="docker-compose-dev.yml"
fi

# Deploy with Docker Compose
docker-compose -f $COMPOSE_FILE down
docker-compose -f $COMPOSE_FILE up -d --build

# Check status
docker-compose -f $COMPOSE_FILE ps
