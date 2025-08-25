#!/usr/bin/env bash
# Judge0 Resource User Messages
# This file defines all user-facing messages for better UX

# ============================================================================
# INSTALLATION MESSAGES
# ============================================================================
export JUDGE0_MSG_INSTALL_START="🚀 Installing Judge0 code execution service..."
export JUDGE0_MSG_INSTALL_CHECKING="📋 Checking system requirements..."
export JUDGE0_MSG_INSTALL_DOCKER="🐳 Setting up Docker containers..."
export JUDGE0_MSG_INSTALL_NETWORK="🔗 Creating isolated network..."
export JUDGE0_MSG_INSTALL_VOLUME="💾 Creating data volume..."
export JUDGE0_MSG_INSTALL_API_KEY="🔐 Generating secure API key..."
export JUDGE0_MSG_INSTALL_CONFIG="⚙️  Writing configuration..."
export JUDGE0_MSG_INSTALL_STARTING="▶️  Starting Judge0 services..."
export JUDGE0_MSG_INSTALL_HEALTH="🏥 Waiting for health check..."
export JUDGE0_MSG_INSTALL_SUCCESS="✅ Judge0 installed successfully!"
export JUDGE0_MSG_INSTALL_FAILED="❌ Judge0 installation failed"

# ============================================================================
# STATUS MESSAGES
# ============================================================================
export JUDGE0_MSG_STATUS_CHECKING="🔍 Checking Judge0 status..."
export JUDGE0_MSG_STATUS_RUNNING="✅ Judge0 is running"
export JUDGE0_MSG_STATUS_STOPPED="⏹️  Judge0 is stopped"
export JUDGE0_MSG_STATUS_ERROR="❌ Judge0 is in error state"
export JUDGE0_MSG_STATUS_NOT_INSTALLED="❓ Judge0 is not installed"
export JUDGE0_MSG_STATUS_WORKERS="👷 Workers: %d active"
export JUDGE0_MSG_STATUS_QUEUE="📊 Queue: %d submissions pending"
export JUDGE0_MSG_STATUS_LANGUAGES="🗣️  Languages: %d available"

# ============================================================================
# OPERATION MESSAGES
# ============================================================================
export JUDGE0_MSG_START="▶️  Starting Judge0..."
export JUDGE0_MSG_STOP="⏹️  Stopping Judge0..."
export JUDGE0_MSG_RESTART="🔄 Restarting Judge0..."
export JUDGE0_MSG_LOGS="📜 Fetching Judge0 logs..."
export JUDGE0_MSG_UNINSTALL="🗑️  Uninstalling Judge0..."

# ============================================================================
# API MESSAGES
# ============================================================================
export JUDGE0_MSG_API_TEST="🧪 Testing Judge0 API..."
export JUDGE0_MSG_API_SUCCESS="✅ API is responding"
export JUDGE0_MSG_API_FAILED="❌ API is not responding"
export JUDGE0_MSG_API_AUTH_FAILED="🔒 API authentication failed"
export JUDGE0_MSG_API_SUBMISSION="📤 Submitting code..."
export JUDGE0_MSG_API_RESULT="📥 Fetching results..."

# ============================================================================
# LANGUAGE MESSAGES
# ============================================================================
export JUDGE0_MSG_LANG_LIST="📋 Available programming languages:"
export JUDGE0_MSG_LANG_INSTALL="📦 Installing language support..."
export JUDGE0_MSG_LANG_NOT_FOUND="❌ Language not supported: %s"

# ============================================================================
# SECURITY MESSAGES
# ============================================================================
export JUDGE0_MSG_SEC_LIMITS="🛡️  Resource limits enforced:"
export JUDGE0_MSG_SEC_CPU="⏱️  CPU time: %ds"
export JUDGE0_MSG_SEC_MEMORY="💾 Memory: %dMB"
export JUDGE0_MSG_SEC_NETWORK="🌐 Network: Disabled"
export JUDGE0_MSG_SEC_SANDBOX="📦 Sandboxing: Enabled"

# ============================================================================
# ERROR MESSAGES
# ============================================================================
export JUDGE0_MSG_ERR_DOCKER="❌ Docker is not running"
export JUDGE0_MSG_ERR_PORT="❌ Port ${JUDGE0_PORT} is already in use"
export JUDGE0_MSG_ERR_PERMISSION="❌ Permission denied. Try with sudo?"
export JUDGE0_MSG_ERR_NETWORK="❌ Network error"
export JUDGE0_MSG_ERR_TIMEOUT="⏰ Operation timed out"
export JUDGE0_MSG_ERR_API_KEY="❌ Invalid API key"
export JUDGE0_MSG_ERR_SUBMISSION="❌ Code submission failed"
export JUDGE0_MSG_ERR_COMPILE="❌ Compilation error"
export JUDGE0_MSG_ERR_RUNTIME="❌ Runtime error"
export JUDGE0_MSG_ERR_MEMORY_LIMIT="❌ Memory limit exceeded"
export JUDGE0_MSG_ERR_TIME_LIMIT="❌ Time limit exceeded"

# ============================================================================
# WARNING MESSAGES
# ============================================================================
export JUDGE0_MSG_WARN_RESOURCES="⚠️  Judge0 requires at least 2GB RAM and 2 CPU cores"
export JUDGE0_MSG_WARN_DISK="⚠️  Low disk space. Judge0 needs at least 5GB free"
export JUDGE0_MSG_WARN_SECURITY="⚠️  Running untrusted code. Ensure proper isolation"
export JUDGE0_MSG_WARN_UPDATE="⚠️  New Judge0 version available: %s"

# ============================================================================
# INFO MESSAGES
# ============================================================================
export JUDGE0_MSG_INFO_DOCS="📚 Documentation: https://judge0.com/docs"
export JUDGE0_MSG_INFO_API="🔗 API endpoint: ${JUDGE0_BASE_URL}"
export JUDGE0_MSG_INFO_DASHBOARD="📊 System info: ${JUDGE0_BASE_URL}/system_info"
export JUDGE0_MSG_INFO_EXAMPLES="💡 Examples: resources/judge0/examples/"

# ============================================================================
# USAGE MESSAGES
# ============================================================================
export JUDGE0_MSG_USAGE_CPU="🖥️  CPU usage: %s%%"
export JUDGE0_MSG_USAGE_MEMORY="💾 Memory usage: %s"
export JUDGE0_MSG_USAGE_SUBMISSIONS="📊 Total submissions: %d"
export JUDGE0_MSG_USAGE_SUCCESS_RATE="✅ Success rate: %s%%"

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

#######################################
# Export all Judge0 messages
#######################################
judge0::export_messages() {
    # Installation
    export JUDGE0_MSG_INSTALL_START
    export JUDGE0_MSG_INSTALL_CHECKING
    export JUDGE0_MSG_INSTALL_DOCKER
    export JUDGE0_MSG_INSTALL_NETWORK
    export JUDGE0_MSG_INSTALL_VOLUME
    export JUDGE0_MSG_INSTALL_API_KEY
    export JUDGE0_MSG_INSTALL_CONFIG
    export JUDGE0_MSG_INSTALL_STARTING
    export JUDGE0_MSG_INSTALL_HEALTH
    export JUDGE0_MSG_INSTALL_SUCCESS
    export JUDGE0_MSG_INSTALL_FAILED
    
    # Status
    export JUDGE0_MSG_STATUS_CHECKING
    export JUDGE0_MSG_STATUS_RUNNING
    export JUDGE0_MSG_STATUS_STOPPED
    export JUDGE0_MSG_STATUS_ERROR
    export JUDGE0_MSG_STATUS_NOT_INSTALLED
    export JUDGE0_MSG_STATUS_WORKERS
    export JUDGE0_MSG_STATUS_QUEUE
    export JUDGE0_MSG_STATUS_LANGUAGES
    
    # Operations
    export JUDGE0_MSG_START
    export JUDGE0_MSG_STOP
    export JUDGE0_MSG_RESTART
    export JUDGE0_MSG_LOGS
    export JUDGE0_MSG_UNINSTALL
    
    # API
    export JUDGE0_MSG_API_TEST
    export JUDGE0_MSG_API_SUCCESS
    export JUDGE0_MSG_API_FAILED
    export JUDGE0_MSG_API_AUTH_FAILED
    export JUDGE0_MSG_API_SUBMISSION
    export JUDGE0_MSG_API_RESULT
    
    # Languages
    export JUDGE0_MSG_LANG_LIST
    export JUDGE0_MSG_LANG_INSTALL
    export JUDGE0_MSG_LANG_NOT_FOUND
    
    # Security
    export JUDGE0_MSG_SEC_LIMITS
    export JUDGE0_MSG_SEC_CPU
    export JUDGE0_MSG_SEC_MEMORY
    export JUDGE0_MSG_SEC_NETWORK
    export JUDGE0_MSG_SEC_SANDBOX
    
    # Errors
    export JUDGE0_MSG_ERR_DOCKER
    export JUDGE0_MSG_ERR_PORT
    export JUDGE0_MSG_ERR_PERMISSION
    export JUDGE0_MSG_ERR_NETWORK
    export JUDGE0_MSG_ERR_TIMEOUT
    export JUDGE0_MSG_ERR_API_KEY
    export JUDGE0_MSG_ERR_SUBMISSION
    export JUDGE0_MSG_ERR_COMPILE
    export JUDGE0_MSG_ERR_RUNTIME
    export JUDGE0_MSG_ERR_MEMORY_LIMIT
    export JUDGE0_MSG_ERR_TIME_LIMIT
    
    # Warnings
    export JUDGE0_MSG_WARN_RESOURCES
    export JUDGE0_MSG_WARN_DISK
    export JUDGE0_MSG_WARN_SECURITY
    export JUDGE0_MSG_WARN_UPDATE
    
    # Info
    export JUDGE0_MSG_INFO_DOCS
    export JUDGE0_MSG_INFO_API
    export JUDGE0_MSG_INFO_DASHBOARD
    export JUDGE0_MSG_INFO_EXAMPLES
    
    # Usage
    export JUDGE0_MSG_USAGE_CPU
    export JUDGE0_MSG_USAGE_MEMORY
    export JUDGE0_MSG_USAGE_SUBMISSIONS
    export JUDGE0_MSG_USAGE_SUCCESS_RATE
}