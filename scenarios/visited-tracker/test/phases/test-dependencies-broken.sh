#!/bin/bash
# Dependencies validation phase - <30 seconds
# Validates required resources and service dependencies
set -uo pipefail

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

error_count=0
warning_count=0

# Helper function to check URL health
check_url_health() {
    local url="$1"
    local timeout="${2:-5}"
    
    if timeout "$timeout" curl -sf "$url" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check PostgreSQL availability (required dependency)
echo "🔍 Checking PostgreSQL (required)..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL is running and ready${NC}"
    else
        echo -e "${RED}❌ PostgreSQL smoke test failed${NC}"
        echo "   Start with: vrooli resource start postgres"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ resource-postgres CLI not available${NC}"
    echo "   Install with: vrooli setup"
    ((error_count++))
fi

# Check Redis availability (optional dependency)
echo "🔍 Checking Redis (optional)..."
if command -v resource-redis >/dev/null 2>&1; then
    if resource-redis test smoke >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis is running and responsive${NC}"
    else
        echo -e "${YELLOW}⚠️  Redis smoke test failed (optional dependency)${NC}"
        echo "   Start with: vrooli resource start redis"
        ((warning_count++))
    fi
else
    echo -e "${YELLOW}⚠️  resource-redis CLI not available (optional dependency)${NC}"
    echo "   Install with: vrooli setup"
    ((warning_count++))
fi

# Check Go environment
echo "🔍 Checking Go environment..."
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
    echo -e "${GREEN}✅ Go is available: $go_version${NC}"
    
    # Check if Go modules can be downloaded (basic connectivity)
    cd api
    if go mod download >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Go module dependencies can be downloaded${NC}"
    else
        echo -e "${RED}❌ Go module dependencies download failed${NC}"
        echo "   Check internet connectivity or module cache"
        ((error_count++))
    fi
    cd ..
else
    echo -e "${RED}❌ Go is not installed${NC}"
    echo "   Install Go: https://golang.org/doc/install"
    ((error_count++))
fi

# Check Node.js environment
echo "🔍 Checking Node.js environment..."
if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    echo -e "${GREEN}✅ Node.js is available: $node_version${NC}"
    
    if command -v npm >/dev/null 2>&1; then
        npm_version=$(npm --version)
        echo -e "${GREEN}✅ npm is available: v$npm_version${NC}"
        
        # Check if npm can install dependencies
        cd ui
        if [ -f "package.json" ]; then
            # Don't actually install, just check if package.json is valid
            set +e  # Temporarily disable exit on error
            npm list >/dev/null 2>&1
            npm_exit_code=$?
            set -e  # Re-enable exit on error
            
            if [ $npm_exit_code -eq 0 ] || [ $npm_exit_code -eq 1 ]; then  # npm list exits with 1 for missing deps
                echo -e "${GREEN}✅ Node.js dependencies structure valid${NC}"
            else
                echo -e "${RED}❌ Node.js dependencies have issues${NC}"
                echo "   Run: cd ui && npm install"
                ((error_count++))
            fi
        fi
        cd ..
    else
        echo -e "${RED}❌ npm is not available${NC}"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ Node.js is not installed${NC}"
    echo "   Install Node.js: https://nodejs.org/"
    ((error_count++))
fi

# Check required tools
echo "🔍 Checking required tools..."
required_tools=("jq" "curl")
optional_tools=("nc" "redis-cli" "pg_isready")

for tool in "${required_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $tool is available${NC}"
    else
        echo -e "${RED}❌ $tool is not installed (required)${NC}"
        case "$tool" in
            jq) echo "   Install: sudo apt-get install jq" ;;
            curl) echo "   Install: sudo apt-get install curl" ;;
        esac
        ((error_count++))
    fi
done

for tool in "${optional_tools[@]}"; do
    # Use which instead of command -v for optional tools to avoid hanging
    if which "$tool" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $tool is available${NC}"
    else
        echo -e "${YELLOW}⚠️  $tool is not installed (recommended)${NC}"
        ((warning_count++))
    fi
done

# Check environment variables
echo "🔍 Checking environment setup..."
if [ -n "${VROOLI_ROOT:-}" ]; then
    echo -e "${GREEN}✅ VROOLI_ROOT is set: $VROOLI_ROOT${NC}"
else
    echo -e "${YELLOW}⚠️  VROOLI_ROOT not set (may use defaults)${NC}"
    ((warning_count++))
fi

# Check port availability (dynamic port allocation)
echo "🔍 Checking port registry..."
if [ -f "${VROOLI_ROOT:-}/scripts/resources/port-registry.sh" ]; then
    echo -e "${GREEN}✅ Port registry available for dynamic allocation${NC}"
else
    echo -e "${YELLOW}⚠️  Port registry not found (may use defaults)${NC}"
    ((warning_count++))
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        echo -e "${GREEN}✅ Dependencies validation completed successfully in ${duration}s${NC}"
    else
        echo -e "${GREEN}✅ Dependencies validation completed with $warning_count warnings in ${duration}s${NC}"
    fi
else
    echo -e "${RED}❌ Dependencies validation failed with $error_count errors and $warning_count warnings in ${duration}s${NC}"
fi

if [ $duration -gt 30 ]; then
    echo -e "${YELLOW}⚠️  Dependencies phase exceeded 30s target${NC}"
fi

# Show summary of what to do next
if [ $error_count -gt 0 ]; then
    echo ""
    echo -e "${BLUE}💡 To fix dependency issues:${NC}"
    echo "   1. Start required resources: vrooli resource start postgres"
    echo "   2. Install missing tools (see specific instructions above)"
    echo "   3. Check internet connectivity for module downloads"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi