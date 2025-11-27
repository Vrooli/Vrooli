#!/usr/bin/env bash
set -euo pipefail

# Optional helper to fetch the Chromium bundle Playwright expects.
# Skip if you plan to reuse an existing Chromium via PLAYWRIGHT_CHROMIUM_PATH.
# If @playwright/browser-chromium is present, prefer that (offline-friendly).

set +e
VENDORED_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/node_modules/@playwright/browser-chromium"
set -e

if [[ -d "${VENDORED_PATH}" ]]; then
  echo "Playwright browser already vendored at ${VENDORED_PATH}; skipping download"
  exit 0
fi

if ! command -v npx >/dev/null 2>&1; then
  echo "npx not found; skipping Playwright browser install (set PLAYWRIGHT_CHROMIUM_PATH or vendor @playwright/browser-chromium)" >&2
  exit 0
fi

echo "Installing Playwright Chromium bundle via npx (no vendor found)..."
npx playwright install chromium
