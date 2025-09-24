#!/bin/bash
################################################################################
# Test Genie Environment Setup Script
# 
# This script sets up the environment for running test-genie with Vrooli's
# shared postgres resource. It handles dynamic port allocation and credential
# discovery.
################################################################################

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Setting up Test Genie environment...${NC}"

# Get postgres credentials from docker
POSTGRES_PASSWORD=$(docker exec vrooli-postgres-main printenv POSTGRES_PASSWORD 2>/dev/null || echo "")
if [[ -z "$POSTGRES_PASSWORD" ]]; then
    echo -e "${RED}❌ Could not retrieve postgres password from container${NC}"
    echo "   Please ensure postgres resource is running: resource-postgres status"
    exit 1
fi

# Find an available port for UI (avoiding conflicts)
find_available_port() {
    local start_port=$1
    local end_port=$2
    
    for port in $(seq $start_port $end_port); do
        if ! lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo $port
            return 0
        fi
    done
    return 1
}

# Check if port 3000 is in use
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Port 3000 is in use, finding alternative...${NC}"
    UI_PORT=$(find_available_port 3050 3099)
    if [[ -z "$UI_PORT" ]]; then
        echo -e "${RED}❌ No available ports in range 3050-3099${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Using port $UI_PORT for UI${NC}"
else
    UI_PORT=3000
fi

# Create .env file
cat > .env << EOF
# Test Genie Environment Configuration
# Generated: $(date)

# API Configuration
API_PORT=8200

# UI Configuration  
UI_PORT=$UI_PORT

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_USER=vrooli
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
POSTGRES_DB=test_genie

# OpenCode Model Configuration
OPENCODE_DEFAULT_MODEL=openrouter/x-ai/grok-code-fast-1
TEST_GENIE_AGENT_MODEL=${TEST_GENIE_AGENT_MODEL:-openrouter/x-ai/grok-code-fast-1}

# Optional: Redis Configuration
# REDIS_HOST=localhost
# REDIS_PORT=6379

# Optional: Qdrant Configuration
# QDRANT_HOST=localhost
# QDRANT_PORT=6333
EOF

echo -e "${GREEN}✅ Environment file created: .env${NC}"

# Create database if it doesn't exist
echo -e "${BLUE}📊 Ensuring test_genie database exists...${NC}"
docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "CREATE DATABASE test_genie;" 2>/dev/null || echo "   Database already exists"

# Initialize schema if needed
if [[ -f "initialization/storage/postgres/schema.sql" ]]; then
    echo -e "${BLUE}🗄️  Initializing database schema...${NC}"
    # Note: This would normally use resource-postgres or direct psql, simplified for now
    echo "   Schema initialization would be run here"
fi

echo -e "${GREEN}✅ Environment setup complete!${NC}"
echo ""
echo -e "${CYAN}📋 Configuration Summary:${NC}"
echo "   API Port: 8200"
echo "   UI Port: $UI_PORT"
echo "   Database: test_genie@localhost:5433"
echo ""
echo -e "${YELLOW}💡 To start test-genie:${NC}"
echo "   source .env"
echo "   make run"
echo ""
echo -e "${YELLOW}💡 Or manually:${NC}"
echo "   export \$(cat .env | grep -v '^#' | xargs)"
echo "   cd api && ./test-genie-api &"
echo "   cd ui && npm start &"
