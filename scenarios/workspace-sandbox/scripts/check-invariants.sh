#!/usr/bin/env bash
# check-invariants.sh — fail-loud CI gate that ensures every invariant
# ID listed in docs/internal/INVARIANTS.md has a matching t.Run("I-...")
# subtest somewhere in scenarios/workspace-sandbox/api/internal/.
#
# Usage:
#   bash scripts/check-invariants.sh
#
# Exit codes:
#   0 — every documented ID has a test
#   1 — at least one ID is missing a t.Run; the missing IDs are printed
#
# This gate is intentionally simple: a `grep` of test files for each
# documented ID. It does NOT execute tests — it only checks that they
# exist. The full assertion lives inside the test bodies.

set -euo pipefail

# Resolve paths relative to the workspace-sandbox scenario root.
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
scenario_root="$(cd "$script_dir/.." && pwd)"
doc="$scenario_root/docs/internal/INVARIANTS.md"
tests_root="$scenario_root/api/internal"

if [[ ! -f "$doc" ]]; then
  echo "check-invariants: INVARIANTS.md not found at $doc" >&2
  exit 1
fi
if [[ ! -d "$tests_root" ]]; then
  echo "check-invariants: tests root not found at $tests_root" >&2
  exit 1
fi

# Extract every "### I-XXX-N" header from the doc.
mapfile -t ids < <(grep -oE '^### I-[A-Z]+-[0-9]+' "$doc" | awk '{print $2}')

if [[ ${#ids[@]} -eq 0 ]]; then
  echo "check-invariants: no invariant IDs found in $doc" >&2
  exit 1
fi

missing=()
for id in "${ids[@]}"; do
  # Match the literal "I-XXX-N" string in any test file. The scan is
  # intentionally permissive: t.Run("I-...") direct, table-driven
  # entries `{name: "I-...", ...}` and TestI_XXX_N function names all
  # qualify, since they all surface the ID in the test runner output.
  if ! grep -rqF "\"$id\"" --include='*_test.go' "$tests_root" \
     && ! grep -rqE "func Test[A-Za-z0-9_]*$(echo "$id" | tr '-' '_')" --include='*_test.go' "$tests_root"; then
    missing+=("$id")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "check-invariants: the following invariant IDs are documented in INVARIANTS.md but have no matching test:" >&2
  for id in "${missing[@]}"; do
    echo "  - $id" >&2
  done
  echo "" >&2
  echo "Add a 't.Run(\"<ID>\", ...)' subtest in the appropriate package's invariants_test.go." >&2
  exit 1
fi

echo "check-invariants: ${#ids[@]} invariants documented, all have matching tests."
