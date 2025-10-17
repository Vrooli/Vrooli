#!/bin/bash

set -e

echo "=== Dependency Tests ==="

# Test required resources
echo "Testing required resources..."

# PostgreSQL
echo "Checking PostgreSQL..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres status >/dev/null 2>&1; then
        echo "✅ PostgreSQL available"
    else
        echo "⚠️  PostgreSQL not running (required for full functionality)"
    fi
else
    echo "❌ PostgreSQL resource not found"
    exit 1
fi

# Home Assistant (optional but important)
echo "Checking Home Assistant..."
if command -v resource-home-assistant >/dev/null 2>&1; then
    if resource-home-assistant status >/dev/null 2>&1; then
        echo "✅ Home Assistant available"
    else
        echo "⚠️  Home Assistant not running (will use mock mode)"
    fi
else
    echo "⚠️  Home Assistant resource not installed (will use mock mode)"
fi

# Scenario Authenticator (required)
echo "Checking Scenario Authenticator..."
if command -v scenario-authenticator >/dev/null 2>&1; then
    echo "✅ Scenario Authenticator CLI found"
else
    echo "⚠️  Scenario Authenticator CLI not found"
fi

# Calendar (optional)
echo "Checking Calendar service..."
if command -v calendar >/dev/null 2>&1; then
    echo "✅ Calendar CLI found"
else
    echo "⚠️  Calendar service not installed (will use fallback scheduling)"
fi

# Claude Code (required for AI generation)
echo "Checking Claude Code..."
if command -v resource-claude-code >/dev/null 2>&1; then
    echo "✅ Claude Code resource found"
else
    echo "⚠️  Claude Code resource not found (AI generation will be limited)"
fi

# Redis (optional)
echo "Checking Redis..."
if command -v resource-redis >/dev/null 2>&1; then
    if resource-redis status >/dev/null 2>&1; then
        echo "✅ Redis available"
    else
        echo "⚠️  Redis not running (will use in-memory caching)"
    fi
else
    echo "⚠️  Redis not installed (will use in-memory caching)"
fi

# Check Go installation
echo "Checking Go installation..."
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | awk '{print $3}')
    echo "✅ Go installed: $go_version"
else
    echo "❌ Go not found"
    exit 1
fi

# Check Node.js for UI
echo "Checking Node.js installation..."
if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    echo "✅ Node.js installed: $node_version"
else
    echo "❌ Node.js not found"
    exit 1
fi

echo "✅ Dependency tests completed"
