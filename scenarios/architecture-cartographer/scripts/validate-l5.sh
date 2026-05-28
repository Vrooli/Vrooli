#!/usr/bin/env bash
# validate-l5.sh — live, end-to-end smoke test for the cartographer's
# L5 programmatic drift checks. Exits non-zero on the first invariant
# violation; prints which one. Intended to run against a live, healthy
# architecture-cartographer scenario.
#
# Usage:
#   ./scripts/validate-l5.sh                 # uses architecture-cartographer
#   ./scripts/validate-l5.sh swarm-manager   # validates against another scenario
#
# Invariants enforced (each maps to a Section 10 / Target End State item
# in plan cartographer-revive-detectors-fix-signals-score-ux-…):
#   - signals boundaries returns at least one non-zero coupling row
#   - conflicts detect returns at least one finding (real drift OR the
#     authority_fallback info finding)
#   - signals score by repo-relative path succeeds
#   - domains extract surfaces "Authority confidence:" in its output

set -euo pipefail

SCENARIO="${1:-architecture-cartographer}"
PROBE_FILE="${2:-api/internal/graph/types.go}"

CLI=architecture-cartographer

step() { printf '\n[validate-l5] %s\n' "$*"; }
die()  { printf '\n[validate-l5] FAIL: %s\n' "$*" >&2; exit 1; }

step "signals boundaries ${SCENARIO}"
boundaries_out=$("${CLI}" signals boundaries "${SCENARIO}")
printf '%s\n' "${boundaries_out}" | tail -n 20
if ! printf '%s\n' "${boundaries_out}" | awk '
  /Ce=[0-9]+/ {
    for (i = 1; i <= NF; i++)
      if ($i ~ /^Ce=/) {
        split($i, a, "="); if (a[2]+0 > 0) { found=1 }
      }
  }
  END { exit found ? 0 : 1 }
'; then
  die "boundaries report has Ce=0 for every domain (the broken-detector regression)"
fi

step "conflicts detect ${SCENARIO}"
conflicts_out=$("${CLI}" conflicts detect "${SCENARIO}")
printf '%s\n' "${conflicts_out}" | tail -n 20
if printf '%s\n' "${conflicts_out}" | grep -q "Detected 0 conflict"; then
  die "conflicts detect returned 0 — at minimum the authority_fallback info finding should fire when DOMAINS.md is missing"
fi

step "signals score ${SCENARIO} ${PROBE_FILE}"
score_out=$("${CLI}" signals score "${SCENARIO}" "${PROBE_FILE}")
printf '%s\n' "${score_out}" | head -n 20
if ! printf '%s\n' "${score_out}" | grep -qE "tier="; then
  die "signals score by path did not produce a verdict line (look for 'tier=')"
fi

step "domains extract ${SCENARIO}"
extract_out=$("${CLI}" domains extract "${SCENARIO}")
printf '%s\n' "${extract_out}" | head -n 10
if ! printf '%s\n' "${extract_out}" | grep -q "Authority confidence:"; then
  die "domains extract did not surface 'Authority confidence:' line"
fi

step "all invariants hold for ${SCENARIO}"
