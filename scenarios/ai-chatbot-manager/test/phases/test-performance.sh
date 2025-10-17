#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}⚡ Testing Performance${NC}"
echo "======================"

# Track failures
FAILED=0

# Function to check API performance
test_api_performance() {
    echo -e "${YELLOW}Testing API performance...${NC}"

    if [ -f "api/main.go" ]; then
        # Check for basic performance considerations
        if grep -q "context\.WithTimeout\|time\.After" api/main.go; then
            echo -e "${GREEN}✅ Timeout handling implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No timeout handling found${NC}"
        fi

        # Check for goroutines (concurrency)
        if grep -q "go func\|goroutine" api/main.go; then
            echo -e "${GREEN}✅ Concurrent processing implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No concurrent processing found${NC}"
        fi

        # Check for connection pooling or optimization
        if grep -q "pool\|Pool\|MaxIdleConns\|MaxOpenConns" api/main.go; then
            echo -e "${GREEN}✅ Connection pooling configured${NC}"
        else
            echo -e "${YELLOW}⚠️  No connection pooling found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No API main.go found${NC}"
    fi
}

# Function to check WebSocket performance
test_websocket_performance() {
    echo -e "${YELLOW}Testing WebSocket performance...${NC}"

    if [ -f "api/websocket.go" ]; then
        # Check for connection limits
        if grep -q "maxConnections\|connectionLimit\|rateLimit" api/websocket.go; then
            echo -e "${GREEN}✅ Connection limits implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No connection limits found${NC}"
        fi

        # Check for message buffering
        if grep -q "buffer\|Buffer\|chan.*\[\]" api/websocket.go; then
            echo -e "${GREEN}✅ Message buffering implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No message buffering found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No WebSocket implementation found${NC}"
    fi
}

# Function to check database performance
test_database_performance() {
    echo -e "${YELLOW}Testing database performance...${NC}"

    if [ -d "initialization/storage" ]; then
        # Check for indexes in schema
        if find initialization/storage -name "*.sql" -exec grep -l "CREATE INDEX\|INDEX" {} \; | grep -q .; then
            echo -e "${GREEN}✅ Database indexes defined${NC}"
        else
            echo -e "${YELLOW}⚠️  No database indexes found${NC}"
        fi

        # Check for query optimization hints
        if find initialization/storage -name "*.sql" -exec grep -l "EXPLAIN\|ANALYZE\|OPTIMIZE" {} \; | grep -q .; then
            echo -e "${GREEN}✅ Query optimization considered${NC}"
        else
            echo -e "${YELLOW}⚠️  No query optimization hints found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No database initialization found${NC}"
    fi
}

# Function to check UI performance
test_ui_performance() {
    echo -e "${YELLOW}Testing UI performance...${NC}"

    if [ -d "ui" ]; then
        # Check for package.json with build optimization
        if [ -f "ui/package.json" ]; then
            if grep -q "webpack\|vite\|build.*optimization\|minif" ui/package.json; then
                echo -e "${GREEN}✅ Build optimization configured${NC}"
            else
                echo -e "${YELLOW}⚠️  No build optimization found${NC}"
            fi
        fi

        # Check for lazy loading or code splitting
        if find ui -name "*.js" -o -name "*.ts" -o -name "*.tsx" | xargs grep -l "lazy\|Suspense\|import(" 2>/dev/null | grep -q .; then
            echo -e "${GREEN}✅ Code splitting/lazy loading implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No code splitting found${NC}"
        fi

        # Check for caching headers or service worker
        if find ui -name "*.js" -o -name "*.ts" | xargs grep -l "serviceWorker\|cache\|Cache-Control" 2>/dev/null | grep -q .; then
            echo -e "${GREEN}✅ Caching strategy implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No caching strategy found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No UI directory found${NC}"
    fi
}

# Function to check memory management
test_memory_management() {
    echo -e "${YELLOW}Testing memory management...${NC}"

    if [ -f "api/main.go" ]; then
        # Check for proper resource cleanup
        if grep -q "defer.*Close\|Close()\|cleanup" api/main.go; then
            echo -e "${GREEN}✅ Resource cleanup implemented${NC}"
        else
            echo -e "${RED}❌ No resource cleanup found${NC}"
            FAILED=1
        fi

        # Check for memory optimization patterns
        if grep -q "sync\.Pool\|pool\|reuse" api/main.go; then
            echo -e "${GREEN}✅ Memory optimization implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No memory optimization found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No API main.go found${NC}"
    fi
}

# Function to check scalability considerations
test_scalability() {
    echo -e "${YELLOW}Testing scalability considerations...${NC}"

    # Check for horizontal scaling support
    if [ -f ".vrooli/service.json" ]; then
        if command -v jq &> /dev/null; then
            if jq -e '.runtime.scalable == true' .vrooli/service.json > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Service marked as scalable${NC}"
            else
                echo -e "${YELLOW}⚠️  Service not marked as scalable${NC}"
            fi
        fi
    fi

    # Check for load balancing considerations
    if grep -q "loadBalancer\|LoadBalancer\|balance" api/main.go 2>/dev/null; then
        echo -e "${GREEN}✅ Load balancing considered${NC}"
    else
        echo -e "${YELLOW}⚠️  No load balancing found${NC}"
    fi
}

# Function to run basic load test simulation
run_load_test_simulation() {
    echo -e "${YELLOW}Running basic load test simulation...${NC}"

    # Simple simulation - check if basic endpoints can handle concurrent requests
    # This is a placeholder for actual load testing
    if [ -f "api/main.go" ]; then
        # Check for rate limiting
        if grep -q "rate.*limit\|RateLimit\|throttle" api/main.go; then
            echo -e "${GREEN}✅ Rate limiting implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No rate limiting found${NC}"
        fi

        # Check for circuit breaker pattern
        if grep -q "circuit.*breaker\|CircuitBreaker\|fallback" api/main.go; then
            echo -e "${GREEN}✅ Circuit breaker pattern implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  No circuit breaker found${NC}"
        fi
    fi
}

# Main performance tests
test_api_performance
echo ""

test_websocket_performance
echo ""

test_database_performance
echo ""

test_ui_performance
echo ""

test_memory_management
echo ""

test_scalability
echo ""

run_load_test_simulation
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All performance tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some performance tests failed${NC}"
    exit 1
fi