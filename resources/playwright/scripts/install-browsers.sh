#!/usr/bin/env bash
set -euo pipefail

# Optional helper to fetch the Chromium bundle Playwright expects. Skip if you
# plan to reuse an existing Chromium via PLAYWRIGHT_CHROMIUM_PATH.
if ! command -v npx >/dev/null 2>&1; then
  echo "npx not found; skipping Playwright browser install" >&2
  exit 0
fi

echo "Installing Playwright Chromium bundle..."
npx playwright install chromium
