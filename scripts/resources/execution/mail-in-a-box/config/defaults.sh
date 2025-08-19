#!/bin/bash

# Default configuration for Mail-in-a-Box resource
# This provides email server capabilities for scenarios

# Service configuration
export MAILINABOX_NAME="mail-in-a-box"
export MAILINABOX_CATEGORY="execution"
export MAILINABOX_DESCRIPTION="All-in-one email server with webmail, contacts, calendar"

# Container configuration
export MAILINABOX_CONTAINER_NAME="mailinabox"
export MAILINABOX_IMAGE="mailinabox/mailinabox:v68"

# Network configuration
export MAILINABOX_PORT_SMTP="25"
export MAILINABOX_PORT_SUBMISSION="587"
export MAILINABOX_PORT_IMAP="143"
export MAILINABOX_PORT_IMAPS="993"
export MAILINABOX_PORT_POP3="110"
export MAILINABOX_PORT_POP3S="995"
export MAILINABOX_PORT_ADMIN="8543"
export MAILINABOX_PORT_HTTPS="443"

# Data directories
export MAILINABOX_DATA_DIR="${HOME}/.mailinabox"
export MAILINABOX_MAIL_DIR="${MAILINABOX_DATA_DIR}/mail"
export MAILINABOX_CONFIG_DIR="${MAILINABOX_DATA_DIR}/config"
export MAILINABOX_SSL_DIR="${MAILINABOX_DATA_DIR}/ssl"

# Default admin email
export MAILINABOX_ADMIN_EMAIL="${MAILINABOX_ADMIN_EMAIL:-admin@mail.local}"
export MAILINABOX_ADMIN_PASSWORD="${MAILINABOX_ADMIN_PASSWORD:-ChangeMe123!}"

# Domain configuration  
export MAILINABOX_PRIMARY_HOSTNAME="${MAILINABOX_PRIMARY_HOSTNAME:-mail.local}"
export MAILINABOX_BIND_ADDRESS="${MAILINABOX_BIND_ADDRESS:-127.0.0.1}"

# Feature flags
export MAILINABOX_ENABLE_WEBMAIL="true"
export MAILINABOX_ENABLE_CALDAV="true"
export MAILINABOX_ENABLE_CARDDAV="true"
export MAILINABOX_ENABLE_AUTOCONFIG="true"

# Security settings
export MAILINABOX_ENABLE_GREYLISTING="true"
export MAILINABOX_ENABLE_SPAMASSASSIN="true"
export MAILINABOX_ENABLE_FAIL2BAN="true"
export MAILINABOX_ENABLE_TLS="true"