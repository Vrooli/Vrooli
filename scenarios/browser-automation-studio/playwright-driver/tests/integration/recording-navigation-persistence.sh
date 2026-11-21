#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DRIVER_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "${DRIVER_DIR}"
pnpm test:e2e:pipeline
