#!/usr/bin/env bash
# Shared install helpers for the coding-agent resources (claude-code, codex,
# opencode). Each installs an upstream CLI into a USER-writable location so
# updates never need root. The install *mechanism* differs per agent
# (codex = npm into a user prefix, claude-code = native installer,
# opencode = binary download); only the path/ownership policy is shared here.
#
# Callers must have sourced scripts/lib/utils/log.sh (for log::*) first.

# agent_install::blocking_system_install <binary> <managed_bin_dir>
# Echoes the path of an existing <binary> on PATH that lives OUTSIDE
# <managed_bin_dir> in a directory we cannot write to (root-owned) — a copy
# that would need sudo to replace and must not be clobbered. Empty output
# means "no blocker": either nothing is installed, or the existing copy is
# already under our managed dir, or its directory is user-writable.
agent_install::blocking_system_install() {
    local binary="$1" managed_bin_dir="$2" existing dir
    existing="$(command -v "${binary}" 2>/dev/null || true)"
    [[ -n "${existing}" ]] || return 0
    case "${existing}" in
        "${managed_bin_dir}/"*) return 0 ;;
    esac
    dir="$(dirname "${existing}")"
    if [[ ! -w "${dir}" ]]; then
        printf '%s' "${existing}"
    fi
}

# agent_install::warn_if_shadowed <binary> <managed_bin_dir>
# Warns when <binary> does not resolve at all, or resolves to a copy outside
# <managed_bin_dir> that precedes our managed install on PATH (shadowing it).
agent_install::warn_if_shadowed() {
    local binary="$1" managed_bin_dir="$2" resolved
    resolved="$(command -v "${binary}" 2>/dev/null || true)"
    if [[ -z "${resolved}" ]]; then
        log::warning "${managed_bin_dir} may not be on PATH — add it so '${binary}' resolves."
        return 0
    fi
    case "${resolved}" in
        "${managed_bin_dir}/"*) return 0 ;;
        *) log::warning "Another '${binary}' at ${resolved} precedes ${managed_bin_dir} on PATH; the user install is shadowed until PATH is reordered (put ${managed_bin_dir} first)." ;;
    esac
}
