#!/bin/bash
# Dependencies validation phase - verifies all dependencies are available and healthy
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 30-second target
testing::phase::init --target-time "30s"

echo "🔍 Checking language dependencies..."

# Check Go
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | awk '{print $3}')
    log::success "✅ Go is available: $go_version"
    testing::phase::add_test passed
else
    testing::phase::add_error "❌ Go is not installed"
    testing::phase::add_test failed
fi

# Check jq (for JSON processing)
if command -v jq >/dev/null 2>&1; then
    jq_version=$(jq --version)
    log::success "✅ jq is available: $jq_version"
    testing::phase::add_test passed
else
    testing::phase::add_warning "⚠️  jq not available (recommended for JSON processing)"
fi

echo ""
echo "🔍 Checking required resources..."

# Check PostgreSQL
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres status >/dev/null 2>&1; then
        log::success "✅ PostgreSQL resource is healthy"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ PostgreSQL resource is not healthy"
        testing::phase::add_test failed
    fi
else
    testing::phase::add_error "❌ PostgreSQL resource CLI not available"
    testing::phase::add_test failed
fi

# Check Ollama
if command -v resource-ollama >/dev/null 2>&1; then
    if resource-ollama status >/dev/null 2>&1; then
        log::success "✅ Ollama resource is healthy"
        testing::phase::add_test passed

        # Check for required models
        echo "  Checking required Ollama models..."
        for model in "llama3.2" "mistral" "nomic-embed-text"; do
            if resource-ollama content list | grep -q "$model"; then
                log::success "  ✅ Model available: $model"
            else
                testing::phase::add_warning "  ⚠️  Model not available: $model (may need to pull)"
            fi
        done
    else
        testing::phase::add_error "❌ Ollama resource is not healthy"
        testing::phase::add_test failed
    fi
else
    testing::phase::add_error "❌ Ollama resource CLI not available"
    testing::phase::add_test failed
fi

# Check Unstructured-io
if command -v resource-unstructured-io >/dev/null 2>&1; then
    if resource-unstructured-io status >/dev/null 2>&1; then
        log::success "✅ Unstructured-io resource is healthy"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Unstructured-io resource is not healthy"
        testing::phase::add_test failed
    fi
else
    testing::phase::add_error "❌ Unstructured-io resource CLI not available"
    testing::phase::add_test failed
fi

# Check N8n (optional but recommended)
if command -v resource-n8n >/dev/null 2>&1; then
    if resource-n8n status >/dev/null 2>&1; then
        log::success "✅ N8n resource is healthy"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  N8n resource is not healthy (optional)"
    fi
else
    testing::phase::add_warning "⚠️  N8n resource CLI not available (optional)"
fi

# Check Qdrant (optional)
if command -v resource-qdrant >/dev/null 2>&1; then
    if resource-qdrant status >/dev/null 2>&1; then
        log::success "✅ Qdrant resource is healthy"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  Qdrant resource is not healthy (optional for semantic search)"
    fi
else
    testing::phase::add_warning "⚠️  Qdrant resource CLI not available (optional)"
fi

# End with summary
testing::phase::end_with_summary "Dependencies validation completed"
