#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BATCH_SIZE="${VITEST_BATCH_SIZE:-1}"
NODE_HEAP_MB="${VITEST_NODE_HEAP_MB:-8192}"

if command -v rg >/dev/null 2>&1; then
  mapfile -t TEST_FILES < <(rg --files src | rg '\.(test|spec)\.(ts|tsx)$' | sort)
else
  mapfile -t TEST_FILES < <(find src -type f \( -name '*.test.ts' -o -name '*.test.tsx' -o -name '*.spec.ts' -o -name '*.spec.tsx' \) | sort)
fi

if [ "${#TEST_FILES[@]}" -eq 0 ]; then
  echo "No Vitest files found under src/"
  exit 1
fi

for ((i = 0; i < ${#TEST_FILES[@]}; i += BATCH_SIZE)); do
  batch=( "${TEST_FILES[@]:i:BATCH_SIZE}" )
  batch_end=$(( i + ${#batch[@]} ))
  echo "[vitest-batch] running files $((i + 1))-$batch_end of ${#TEST_FILES[@]}"
  NODE_OPTIONS="--max-old-space-size=${NODE_HEAP_MB}" pnpm exec vitest run "$@" "${batch[@]}"
done
