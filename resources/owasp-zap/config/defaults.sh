#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Default Configuration
################################################################################

# Service configuration
export ZAP_API_PORT="${ZAP_API_PORT:-8180}"
export ZAP_PROXY_PORT="${ZAP_PROXY_PORT:-8181}"
export ZAP_HOST="${ZAP_HOST:-0.0.0.0}"
export ZAP_CONTAINER_NAME="${ZAP_CONTAINER_NAME:-vrooli-owasp-zap}"
export ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"

# API configuration
export ZAP_API_KEY="${ZAP_API_KEY:-}"  # Generated on first start if empty
export ZAP_DISABLE_API_KEY="${ZAP_DISABLE_API_KEY:-false}"  # Set to true for local dev only

# Scanning configuration
export ZAP_SCAN_TIMEOUT="${ZAP_SCAN_TIMEOUT:-3600}"  # 1 hour default
export ZAP_SPIDER_MAX_DEPTH="${ZAP_SPIDER_MAX_DEPTH:-10}"
export ZAP_SPIDER_MAX_DURATION="${ZAP_SPIDER_MAX_DURATION:-0}"  # 0 = no limit
export ZAP_ATTACK_STRENGTH="${ZAP_ATTACK_STRENGTH:-MEDIUM}"  # LOW, MEDIUM, HIGH, INSANE
export ZAP_ALERT_THRESHOLD="${ZAP_ALERT_THRESHOLD:-MEDIUM}"  # LOW, MEDIUM, HIGH

# Memory configuration
export ZAP_MEMORY="${ZAP_MEMORY:-1g}"  # Docker memory limit
export ZAP_JVM_HEAP="${ZAP_JVM_HEAP:-512m}"  # Java heap size

# Storage configuration
export ZAP_DATA_DIR="${ZAP_DATA_DIR:-${HOME}/.vrooli/resources/owasp-zap}"
export ZAP_SESSION_DIR="${ZAP_SESSION_DIR:-${ZAP_DATA_DIR}/sessions}"
export ZAP_REPORT_DIR="${ZAP_REPORT_DIR:-${ZAP_DATA_DIR}/reports}"
export ZAP_POLICY_DIR="${ZAP_POLICY_DIR:-${ZAP_DATA_DIR}/policies}"

# Timeouts
export ZAP_STARTUP_TIMEOUT="${ZAP_STARTUP_TIMEOUT:-30}"
export ZAP_SHUTDOWN_TIMEOUT="${ZAP_SHUTDOWN_TIMEOUT:-10}"
export ZAP_HEALTH_CHECK_TIMEOUT="${ZAP_HEALTH_CHECK_TIMEOUT:-5}"

# Logging
export ZAP_LOG_LEVEL="${ZAP_LOG_LEVEL:-INFO}"  # DEBUG, INFO, WARN, ERROR
export ZAP_LOG_FILE="${ZAP_LOG_FILE:-${ZAP_DATA_DIR}/zap.log}"