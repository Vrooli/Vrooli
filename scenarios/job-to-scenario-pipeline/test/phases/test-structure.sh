#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api cli ui initialization data)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(Makefile README.md .vrooli/service.json)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

optional_dirs=(tests automation)
for dir in "${optional_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    testing::phase::add_warning "Optional directory '$dir' not present"
  fi
done

testing::phase::end_with_summary "Structure validation completed"
