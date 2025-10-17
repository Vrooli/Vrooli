#!/usr/bin/env bash
# Seed Redis job queues using the managed redis resource CLI

set -euo pipefail

echo "🧰 Initializing Redis job queues..."

if ! command -v resource-redis >/dev/null 2>&1; then
    echo "⚠️  resource-redis CLI not available; skipping queue initialization"
    exit 0
fi

if ! resource-redis status >/dev/null 2>&1; then
    echo "⚠️  Redis resource is not running; skipping queue initialization"
    exit 0
fi

if resource-redis content execute LPUSH scraper:queue:init "ready" >/dev/null 2>&1; then
    echo "✅ Redis job queue seeded"
else
    echo "⚠️  Failed to seed Redis job queue"
fi
