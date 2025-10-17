#!/usr/bin/env bash
# Pandas AI Resource Configuration - v2.0 Universal Contract Compliant

# Resource identification
PANDAS_AI_NAME="pandas-ai"
PANDAS_AI_DESC="AI-powered data analysis and manipulation"
PANDAS_AI_CATEGORY="execution"
PANDAS_AI_VERSION="1.0.0"

# Service configuration
PANDAS_AI_PORT="8095"
PANDAS_AI_HOST="0.0.0.0"

# Directory paths
PANDAS_AI_DATA_DIR="${var_DATA_DIR}/resources/pandas-ai"
PANDAS_AI_VENV_DIR="${PANDAS_AI_DATA_DIR}/venv"
PANDAS_AI_SCRIPTS_DIR="${PANDAS_AI_DATA_DIR}/scripts"
PANDAS_AI_CONTENT_DIR="${PANDAS_AI_DATA_DIR}/content"

# Runtime files
PANDAS_AI_PID_FILE="${PANDAS_AI_DATA_DIR}/pandas-ai.pid"
PANDAS_AI_LOG_FILE="${PANDAS_AI_DATA_DIR}/pandas-ai.log"

# Python configuration
PANDAS_AI_PYTHON_VERSION_MIN="3.8"
PANDAS_AI_PACKAGES="pandasai pandas numpy scikit-learn matplotlib seaborn plotly fastapi uvicorn pydantic"

# Database connectors
PANDAS_AI_DB_PACKAGES="psycopg2-binary redis pymongo"