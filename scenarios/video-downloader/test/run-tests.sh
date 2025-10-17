#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &amp;&amp; pwd)"

cd "$SCRIPT_DIR"

printf "=== Running Video Downloader Tests ===\\n"

./phases/test-structure.sh

./phases/test-dependencies.sh

./phases/test-unit.sh

./phases/test-integration.sh

./phases/test-performance.sh

./phases/test-business.sh

printf "\\n✅ All tests passed!\\n"