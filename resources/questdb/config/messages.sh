#!/usr/bin/env bash
# QuestDB Messages Configuration
# User-facing messages for all operations

#######################################
# Initialize messages
# Idempotent - safe to call multiple times
#######################################
questdb::messages::init() {
    # Status messages
    declare -g -A QUESTDB_STATUS_MESSAGES=(
        ["checking"]="Checking QuestDB status..."
        ["running"]="✅ QuestDB is running on ports HTTP:${QUESTDB_HTTP_PORT}, PG:${QUESTDB_PG_PORT}"
        ["not_running"]="❌ QuestDB is not running"
        ["unhealthy"]="⚠️  QuestDB is running but unhealthy"
        ["starting"]="Starting QuestDB time-series database..."
        ["stopping"]="Stopping QuestDB..."
        ["waiting"]="Waiting for QuestDB to become healthy..."
        ["ready"]="✅ QuestDB is ready for connections"
    )

    # Install messages
    declare -g -A QUESTDB_INSTALL_MESSAGES=(
        ["checking_docker"]="Checking Docker availability..."
        ["creating_directories"]="Creating QuestDB directories..."
        ["pulling_image"]="Pulling QuestDB Docker image..."
        ["creating_network"]="Creating Docker network..."
        ["starting_container"]="Starting QuestDB container..."
        ["initializing"]="Initializing default tables..."
        ["success"]="✅ QuestDB installed successfully"
        ["failed"]="❌ QuestDB installation failed"
    )

    # API messages
    declare -g -A QUESTDB_API_MESSAGES=(
        ["connecting"]="Connecting to QuestDB API..."
        ["executing_query"]="Executing query..."
        ["creating_table"]="Creating table..."
        ["inserting_data"]="Inserting data..."
        ["query_success"]="✅ Query executed successfully"
        ["query_failed"]="❌ Query failed"
        ["table_created"]="✅ Table created successfully"
        ["table_exists"]="⚠️  Table already exists"
        ["connection_failed"]="❌ Failed to connect to QuestDB"
    )

    # Error messages
    declare -g -A QUESTDB_ERROR_MESSAGES=(
        ["docker_not_found"]="❌ Docker is not installed or not running"
        ["port_conflict"]="❌ Port conflict detected. QuestDB ports may be in use"
        ["insufficient_space"]="❌ Insufficient disk space for QuestDB data"
        ["network_error"]="❌ Network error connecting to QuestDB"
        ["permission_denied"]="❌ Permission denied. Check Docker permissions"
        ["timeout"]="❌ Operation timed out"
        ["invalid_query"]="❌ Invalid SQL query syntax"
        ["auth_failed"]="❌ Authentication failed"
    )

    # Info messages
    declare -g -A QUESTDB_INFO_MESSAGES=(
        ["web_console"]="🌐 QuestDB Web Console: ${QUESTDB_BASE_URL}"
        ["pg_connection"]="🔗 PostgreSQL connection: ${QUESTDB_PG_URL}"
        ["api_docs"]="📚 API Documentation: ${QUESTDB_BASE_URL}/docs"
        ["data_dir"]="💾 Data directory: ${QUESTDB_DATA_DIR}"
        ["log_dir"]="📝 Log directory: ${QUESTDB_LOG_DIR}"
        ["performance"]="⚡ Performance: 4M+ rows/sec ingestion"
        ["protocols"]="🔌 Protocols: HTTP REST, PostgreSQL, InfluxDB Line"
    )

    # Export messages
    export QUESTDB_STATUS_MESSAGES QUESTDB_INSTALL_MESSAGES
    export QUESTDB_API_MESSAGES QUESTDB_ERROR_MESSAGES QUESTDB_INFO_MESSAGES
}