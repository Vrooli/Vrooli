#!/usr/bin/env bash
set -euo pipefail

# Root File Cleanup Script
# Organizes non-standard root-level files into appropriate directories

RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"

echo "🧹 Root File Organization"
echo "   Moving non-standard files to appropriate locations"
echo

# Unstructured-io cleanup
if [ -d "$RESOURCES_DIR/ai/unstructured-io" ]; then
    echo "📦 Unstructured-io:"
    cd "$RESOURCES_DIR/ai/unstructured-io"
    
    # Move test scripts to lib/
    if [ -f "test_api.py" ] || [ -f "test-api.sh" ] || [ -f "test-suite.sh" ]; then
        [ -f "test_api.py" ] && mv test_api.py lib/ && echo "   ✓ Moved test_api.py to lib/"
        [ -f "test-api.sh" ] && mv test-api.sh lib/ && echo "   ✓ Moved test-api.sh to lib/"
        [ -f "test-suite.sh" ] && mv test-suite.sh lib/ && echo "   ✓ Moved test-suite.sh to lib/"
    fi
    
    # Move documentation to docs/
    mkdir -p docs
    [ -f "TESTING.md" ] && mv TESTING.md docs/ && echo "   ✓ Moved TESTING.md to docs/"
    [ -f "TROUBLESHOOTING.md" ] && mv TROUBLESHOOTING.md docs/ && echo "   ✓ Moved TROUBLESHOOTING.md to docs/"
    
    # Docker files can stay at root (standard practice)
    echo "   ℹ️  Keeping docker-compose files at root (standard practice)"
fi

# Vault cleanup
if [ -d "$RESOURCES_DIR/storage/vault" ]; then
    echo "📦 Vault:"
    cd "$RESOURCES_DIR/storage/vault"
    
    # Move demo and setup scripts to lib/
    [ -f "demo.sh" ] && mv demo.sh lib/ && echo "   ✓ Moved demo.sh to lib/"
    [ -f "setup-dev-tokens.sh" ] && mv setup-dev-tokens.sh lib/ && echo "   ✓ Moved setup-dev-tokens.sh to lib/"
    [ -f "verify-vault.sh" ] && mv verify-vault.sh lib/ && echo "   ✓ Moved verify-vault.sh to lib/"
    
    # Move documentation to docs/
    mkdir -p docs
    [ -f "QUICKSTART.md" ] && mv QUICKSTART.md docs/ && echo "   ✓ Moved QUICKSTART.md to docs/"
fi

# Postgres cleanup
if [ -d "$RESOURCES_DIR/storage/postgres" ]; then
    echo "📦 Postgres:"
    cd "$RESOURCES_DIR/storage/postgres"
    
    # Move utility scripts to lib/
    [ -f "client-setup.sh" ] && mv client-setup.sh lib/ && echo "   ✓ Moved client-setup.sh to lib/"
    [ -f "resource-monitor.sh" ] && mv resource-monitor.sh lib/ && echo "   ✓ Moved resource-monitor.sh to lib/"
fi

echo
echo "✅ Root file organization complete!"
echo
echo "📊 Standard Structure Achieved:"
echo "   Root level: manage.sh, manage.bats, README.md, inject.sh (if applicable)"
echo "   /lib/: All utility scripts and libraries"
echo "   /config/: Configuration files"
echo "   /docs/: All documentation"
echo "   /docker/: Docker-related files (or docker-compose at root)"
echo "   /examples/: Usage examples (where valuable)"