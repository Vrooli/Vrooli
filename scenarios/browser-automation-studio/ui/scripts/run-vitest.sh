#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

PROJECTS=(
  stores
  components-core
  components-palette
  workflow-builder
)

rm -rf coverage
mkdir -p coverage
export NODE_OPTIONS="${NODE_OPTIONS:---max-old-space-size=8192}"

ARGS=("$@")

for project in "${PROJECTS[@]}"; do
  echo "\n=== Running Vitest project: ${project} ==="
  pnpm exec vitest run --project "$project" "${ARGS[@]}"
  if [ -f coverage/vitest-requirements.json ]; then
    mv coverage/vitest-requirements.json "coverage/vitest-requirements-${project}.json"
  fi
done

node <<'NODE'
const fs = require('fs');
const path = require('path');

const coverageDir = path.join(process.cwd(), 'coverage');
const files = fs.existsSync(coverageDir)
  ? fs.readdirSync(coverageDir).filter((file) => file.startsWith('vitest-requirements-') && file.endsWith('.json'))
  : [];

if (files.length === 0) {
  process.exit(0);
}

const mergeReports = (a, b) => {
  if (!a) return b;
  const merged = new Map();

  const addReq = (req) => {
    const existing = merged.get(req.id);
    if (!existing) {
      merged.set(req.id, { ...req });
      return;
    }
    if (req.status === 'failed') {
      existing.status = 'failed';
    }
    if (req.evidence) {
      existing.evidence = existing.evidence ? `${existing.evidence}; ${req.evidence}` : req.evidence;
    }
    existing.duration_ms = (existing.duration_ms || 0) + (req.duration_ms || 0);
    existing.test_count = (existing.test_count || 0) + (req.test_count || 0);
  };

  a.requirements?.forEach(addReq);
  b.requirements?.forEach(addReq);

  return {
    generated_at: b.generated_at,
    scenario: b.scenario || a.scenario,
    phase: b.phase || a.phase,
    test_framework: b.test_framework || a.test_framework,
    total_tests: (a.total_tests || 0) + (b.total_tests || 0),
    passed_tests: (a.passed_tests || 0) + (b.passed_tests || 0),
    failed_tests: (a.failed_tests || 0) + (b.failed_tests || 0),
    skipped_tests: (a.skipped_tests || 0) + (b.skipped_tests || 0),
    duration_ms: (a.duration_ms || 0) + (b.duration_ms || 0),
    requirements: Array.from(merged.values()).sort((x, y) => x.id.localeCompare(y.id)),
  };
};

let combined = null;
for (const file of files) {
  const report = JSON.parse(fs.readFileSync(path.join(coverageDir, file), 'utf-8'));
  combined = mergeReports(combined, report);
}

if (combined) {
  fs.writeFileSync(path.join(coverageDir, 'vitest-requirements.json'), JSON.stringify(combined, null, 2));
}

for (const file of files) {
  fs.unlinkSync(path.join(coverageDir, file));
}
NODE
