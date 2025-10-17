#!/usr/bin/env bash
# Keycloak Production Configuration
# These settings override defaults when KEYCLOAK_ENV=production

# Security hardening
KEYCLOAK_HOSTNAME_STRICT="true"
KEYCLOAK_HOSTNAME_STRICT_HTTPS="true"
KEYCLOAK_HTTP_ENABLED="false"  # Only HTTPS in production
KEYCLOAK_PROXY="edge"  # Behind a reverse proxy

# Database - PostgreSQL required for production
KEYCLOAK_DB="postgres"

# Performance tuning for production
KEYCLOAK_JVM_OPTS="-Xms1024m -Xmx2048m -XX:+UseG1GC -XX:MaxGCPauseMillis=50"

# Clustering support
KEYCLOAK_CACHE_STACK="kubernetes"  # For Kubernetes deployments
# KEYCLOAK_CACHE_STACK="tcp"  # For standalone cluster

# Logging - less verbose in production
KEYCLOAK_LOG_LEVEL="WARN"

# Features for production
KEYCLOAK_FEATURES="token-exchange,admin-fine-grained-authz,client-policies,ciba,par"

# Session configuration
KEYCLOAK_SESSION_IDLE_TIMEOUT="1800"  # 30 minutes
KEYCLOAK_SESSION_MAX_LIFESPAN="36000"  # 10 hours

# Brute force protection
KEYCLOAK_BRUTE_FORCE_PROTECTED="true"
KEYCLOAK_PERMANENT_LOCKOUT="false"
KEYCLOAK_MAX_LOGIN_FAILURES="5"
KEYCLOAK_WAIT_INCREMENT_SECONDS="60"

# CORS settings
KEYCLOAK_WEB_ORIGINS="+"  # Allow all origins from registered clients

# Rate limiting
KEYCLOAK_RATE_LIMIT_ENABLED="true"
KEYCLOAK_RATE_LIMIT_PER_MINUTE="100"

# Backup configuration
KEYCLOAK_BACKUP_ENABLED="true"
KEYCLOAK_BACKUP_SCHEDULE="0 2 * * *"  # Daily at 2 AM
KEYCLOAK_BACKUP_RETENTION_DAYS="30"
KEYCLOAK_BACKUP_MAX_COUNT="30"