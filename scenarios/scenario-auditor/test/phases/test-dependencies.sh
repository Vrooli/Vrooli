#!/bin/bash

set -e

echo "=== Dependencies Tests ==="

# Check Go dependencies
echo "Checking Go API dependencies..."
cd api
if [ -f "go.mod" ]; then
    echo "Downloading Go dependencies..."
    go mod download
    
    echo "Verifying Go dependencies..."
    go mod verify
    
    echo "Checking for vulnerabilities..."
    if command -v govulncheck >/dev/null 2>&1; then
        govulncheck ./...
        echo "✅ No Go vulnerabilities found"
    else
        echo "⚠️  govulncheck not installed, skipping vulnerability check"
    fi
    
    echo "✅ Go API dependencies verified"
else
    echo "❌ go.mod not found in api directory"
    exit 1
fi
cd ..

echo "Checking Go CLI dependencies..."
cd cli
if [ -f "go.mod" ]; then
    echo "Downloading CLI dependencies..."
    go mod download
    
    echo "Verifying CLI dependencies..."
    go mod verify
    
    echo "✅ Go CLI dependencies verified"
else
    echo "❌ go.mod not found in cli directory"
    exit 1
fi
cd ..

# Check Node.js dependencies
echo "Checking Node.js dependencies..."
cd ui
if [ -f "package.json" ]; then
    if command -v node >/dev/null 2>&1; then
        echo "Node.js version: $(node --version)"
        echo "npm version: $(npm --version)"
        
        if [ ! -d "node_modules" ]; then
            echo "Installing Node.js dependencies..."
            npm install --prefer-offline --no-audit --no-fund --loglevel error
        fi
        
        echo "Checking for security vulnerabilities..."
        npm audit --audit-level moderate || echo "⚠️  Some npm audit issues found"
        
        echo "Checking for outdated packages..."
        npm outdated || echo "⚠️  Some packages may be outdated"
        
        echo "✅ Node.js dependencies verified"
    else
        echo "⚠️  Node.js not installed, skipping UI dependency check"
    fi
else
    echo "❌ package.json not found in ui directory"
    exit 1
fi
cd ..

# Check system dependencies
echo "Checking system dependencies..."

# Check required tools
required_tools=(
    "go"
    "git"
    "curl"
    "jq"
)

for tool in "${required_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        echo "✅ $tool available"
    else
        echo "❌ $tool not found"
        exit 1
    fi
done

# Check optional tools
optional_tools=(
    "node"
    "npm"
    "docker"
)

for tool in "${optional_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        echo "✅ $tool available"
    else
        echo "⚠️  $tool not found (optional)"
    fi
done

# Check Go version
echo "Checking Go version..."
go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
required_go_version="1.21"

if [ "$(printf '%s\n' "$required_go_version" "$go_version" | sort -V | head -n1)" = "$required_go_version" ]; then
    echo "✅ Go version $go_version meets requirement ($required_go_version+)"
else
    echo "❌ Go version $go_version below requirement ($required_go_version+)"
    exit 1
fi

echo "=== Dependencies Tests Complete ==="