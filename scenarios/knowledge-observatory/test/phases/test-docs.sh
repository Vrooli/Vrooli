#!/bin/bash
# Documentation validation phase for knowledge-observatory
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

for doc in README.md PRD.md PROBLEMS.md; do
  testing::phase::check "$doc exists" bash -c "[ -s '$doc' ]"
done

# Basic sanity: README should mention Knowledge Observatory
if testing::phase::check "README names scenario" bash -c "grep -qi 'Knowledge Observatory' README.md"; then
  :
else
  testing::phase::add_warning "README may be missing scenario overview"
fi

# PRD should list requirements
if testing::phase::check "PRD contains Requirements section" bash -c "grep -qi 'Requirements' PRD.md"; then
  :
else
  testing::phase::add_warning "PRD missing explicit Requirements section"
fi

# Ensure docs do not reference removed legacy file
if grep -R "scenario-test.yaml" README.md PRD.md PROBLEMS.md >/dev/null 2>&1; then
  testing::phase::add_warning "Documentation still references legacy scenario-test.yaml"
fi

# Quick link validation for markdown relative paths
broken_refs=0
while IFS= read -r path; do
  clean_path="${path//\`/}"
  if [[ "$clean_path" == http* ]]; then
    continue
  fi
  if [[ "$clean_path" == .* ]] && [ ! -e "$clean_path" ]; then
    testing::phase::add_warning "Broken relative reference: $clean_path"
    broken_refs=$((broken_refs + 1))
  fi
done < <(grep -hoE "\]\(([./][^)]*)\)" README.md PRD.md PROBLEMS.md 2>/dev/null | sed 's/.*(//;s/)//')

if [ "$broken_refs" -eq 0 ]; then
  log::success "No obvious broken documentation links"
fi


testing::phase::end_with_summary "Documentation validation completed"
