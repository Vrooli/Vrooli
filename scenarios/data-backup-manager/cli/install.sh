#!/usr/bin/env bash
set -euo pipefail

script_dir="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
repo_root="$(builtin cd "${script_dir}/../../.." && builtin pwd)"

"${repo_root}/packages/cli-core/install.sh" "scenarios/data-backup-manager/cli" --name "data-backup-manager" --manifest "scenarios/data-backup-manager/.vrooli/service.json"
