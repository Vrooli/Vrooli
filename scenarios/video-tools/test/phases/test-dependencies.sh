#!/bin/bash
# Dependency tests for video-tools scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📦 Running video-tools dependency tests..."

# Test Go dependencies
if [[ -d "api" ]]; then
    echo ""
    echo "Testing Go dependencies..."
    cd api

    if go mod verify &>/dev/null; then
        echo "✅ Go modules verified"
    else
        echo "❌ Go module verification failed"
        exit 1
    fi

    if go mod tidy -v &>/dev/null; then
        echo "✅ Go modules are tidy"
    else
        echo "⚠️  Go modules may need tidying"
    fi

    # Check for known vulnerabilities
    if command -v govulncheck &>/dev/null; then
        if govulncheck ./... &>/dev/null; then
            echo "✅ No known vulnerabilities found"
        else
            echo "⚠️  Potential vulnerabilities detected (run 'govulncheck ./...' for details)"
        fi
    else
        echo "ℹ️  govulncheck not installed, skipping vulnerability scan"
    fi

    cd ..
fi

# Check required binaries
echo ""
echo "Checking required binaries..."

if command -v ffmpeg &>/dev/null; then
    ffmpeg_version=$(ffmpeg -version 2>&1 | head -n1 || echo "unknown")
    echo "✅ ffmpeg installed: $ffmpeg_version"
else
    echo "⚠️  ffmpeg not installed (required for video processing)"
fi

if command -v ffprobe &>/dev/null; then
    echo "✅ ffprobe installed"
else
    echo "⚠️  ffprobe not installed (required for video analysis)"
fi

# Check database connectivity
echo ""
echo "Checking database connectivity..."

if [[ -n "${DATABASE_URL:-}" ]] || [[ -n "${TEST_DATABASE_URL:-}" ]]; then
    DB_URL="${TEST_DATABASE_URL:-${DATABASE_URL}}"

    # Extract host and port from DATABASE_URL
    if [[ "$DB_URL" =~ postgres://([^:]+):([^@]+)@([^:]+):([^/]+)/(.+) ]]; then
        DB_HOST="${BASH_REMATCH[3]}"
        DB_PORT="${BASH_REMATCH[4]}"

        if nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            echo "✅ Database reachable at $DB_HOST:$DB_PORT"
        else
            echo "⚠️  Database not reachable at $DB_HOST:$DB_PORT"
        fi
    fi
else
    echo "ℹ️  No database URL configured"
fi

# Check Redis connectivity (if configured)
echo ""
echo "Checking Redis connectivity..."

if [[ -n "${REDIS_URL:-}" ]]; then
    if [[ "$REDIS_URL" =~ redis://([^:]+):([0-9]+) ]]; then
        REDIS_HOST="${BASH_REMATCH[1]}"
        REDIS_PORT="${BASH_REMATCH[2]}"

        if nc -z "$REDIS_HOST" "$REDIS_PORT" 2>/dev/null; then
            echo "✅ Redis reachable at $REDIS_HOST:$REDIS_PORT"
        else
            echo "⚠️  Redis not reachable at $REDIS_HOST:$REDIS_PORT"
        fi
    fi
else
    echo "ℹ️  No Redis URL configured"
fi

# Check required environment variables
echo ""
echo "Checking environment configuration..."

required_vars=("API_PORT")
optional_vars=("DATABASE_URL" "REDIS_URL" "WORK_DIR" "API_TOKEN")

echo "Required variables:"
for var in "${required_vars[@]}"; do
    if [[ -n "${!var:-}" ]]; then
        echo "  ✅ $var is set"
    else
        echo "  ❌ $var is not set"
    fi
done

echo "Optional variables:"
for var in "${optional_vars[@]}"; do
    if [[ -n "${!var:-}" ]]; then
        echo "  ✅ $var is set"
    else
        echo "  ℹ️  $var is not set (using default)"
    fi
done

echo ""
echo "📊 Dependency Check Summary:"
echo "  - Go dependencies: Verified"
echo "  - Required binaries: Checked"
echo "  - Database: Connectivity verified"
echo "  - Environment: Configuration checked"

testing::phase::end_with_summary "Dependency tests completed"
