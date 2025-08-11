#!/usr/bin/env bash
# n8n Constants and Common Messages
# Centralized strings to reduce duplication

# Error Messages
readonly N8N_ERR_DOCKER_NOT_INSTALLED="Docker is not installed"
readonly N8N_ERR_DOCKER_NOT_RUNNING="Docker daemon is not running"
readonly N8N_ERR_DOCKER_NO_PERMISSION="Current user doesn't have Docker permissions"
readonly N8N_ERR_CONTAINER_NOT_EXISTS="n8n container does not exist"
readonly N8N_ERR_CONTAINER_NOT_RUNNING="n8n is not running"
readonly N8N_ERR_ALREADY_RUNNING="n8n is already running"
readonly N8N_ERR_PORT_IN_USE="Port is already in use"
readonly N8N_ERR_CREATE_DIR_FAILED="Failed to create n8n data directory"
readonly N8N_ERR_API_KEY_MISSING="No API key found"
readonly N8N_ERR_API_KEY_INVALID="API key is invalid or expired"

# Fix Suggestions
readonly N8N_FIX_INSTALL_DOCKER="Please install Docker first: https://docs.docker.com/get-docker/"
readonly N8N_FIX_START_DOCKER="Start Docker with: sudo systemctl start docker"
readonly N8N_FIX_DOCKER_PERMISSION="Add user to docker group: sudo usermod -aG docker \$USER"
readonly N8N_FIX_RUN_INSTALL="Run install first: \$0 --action install"
readonly N8N_FIX_CHECK_LOGS="Check logs: docker logs \$N8N_CONTAINER_NAME"
readonly N8N_FIX_CHECK_PORT="Check if port is in use: lsof -i :\$N8N_PORT"
readonly N8N_FIX_API_SETUP="Set up API key: \$0 --action api-setup"

# Common Patterns
readonly N8N_PATTERN_CONTAINER_CHECK='docker ps --format "{{.Names}}" | grep -q "^\${N8N_CONTAINER_NAME}\$"'
readonly N8N_PATTERN_CONTAINER_EXISTS='docker ps -a --format "{{.Names}}" | grep -q "^\${N8N_CONTAINER_NAME}\$"'