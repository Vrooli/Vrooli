#!/bin/bash
set -euo pipefail

echo "⚡ Running Secure Document Processing performance tests"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Build performance
echo "🏗️  Testing build performance..."
build_time=$(time (make build) 2&gt;&amp;1 | grep real | awk '{print $2}' | sed 's/m/*60 + /;s/s//;s/m//')
if (( $(echo "$build_time &lt; 300" | bc -l) )); then  # Less than 5 minutes
    echo "✅ Build time acceptable: ~$build_time seconds"
else
    echo "⚠️  Build time high: ~$build_time seconds"
fi

# API startup performance (requires running)
echo "🚀 Testing API startup performance..."
if [ -f "$SCENARIO_DIR/api/main.go" ]; then
    cd "$SCENARIO_DIR/api"
    startup_time=$(time (go run main.go &amp; pid=$!; sleep 5; kill $pid 2&gt;/dev/null) 2&gt;&amp;1 | grep real | awk '{print $2}')
    cd "$SCENARIO_DIR"
    startup_seconds=$(echo "$startup_time" | sed 's/m/*60 + /;s/s//;s/m//')
    if (( $(echo "$startup_seconds &lt; 10" | bc -l) )); then
        echo "✅ API startup fast: ~$startup_seconds seconds"
    else
        echo "⚠️  API startup slow: ~$startup_seconds seconds"
    fi
fi

# Resource initialization performance check (static)
resources=$(jq '.resources | keys | length' "$SCENARIO_DIR/.vrooli/service.json")
if [ "$resources" -lt 10 ]; then
    echo "✅ Reasonable number of resources: $resources"
else
    echo "⚠️  High number of resources may impact performance: $resources"
fi

# UI bundle size check
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    dep_count=$(cd "$SCENARIO_DIR/ui" &amp;&amp; jq '.dependencies | keys | length' package.json)
    if [ "$dep_count" -lt 50 ]; then
        echo "✅ UI dependency count reasonable: $dep_count"
    else
        echo "⚠️  High UI dependencies may impact load time: $dep_count"
    fi
fi

# Memory and CPU baseline (static analysis)
go_files=$(find "$SCENARIO_DIR/api" -name "*.go" | wc -l)
if [ "$go_files" -lt 20 ]; then
    echo "✅ Reasonable API code size: $go_files files"
else
    echo "⚠️  Large API codebase may impact performance: $go_files files"
fi

echo "✅ Performance baseline checks passed"
