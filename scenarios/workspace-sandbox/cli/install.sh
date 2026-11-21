#!/usr/bin/env sh
set -eu

prefix=${WORKSPACE_SANDBOX_PREFIX:-${XDG_BIN_HOME:-"$HOME/.local/bin"}}
mkdir -p "$prefix"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
api_dir=$(CDPATH= cd -- "$script_dir/../api" && pwd)

copy_atomic() {
	src=$1
	dst=$2
	tmp="$dst.tmp.$$"
	cp "$src" "$tmp"
	chmod 0755 "$tmp"
	mv "$tmp" "$dst"
}

copy_atomic "$script_dir/workspace-sandbox" "$prefix/workspace-sandbox-cli"
copy_atomic "$api_dir/workspace-sandbox-api" "$prefix/workspace-sandbox-api"
copy_atomic "$script_dir/standalone-launch.sh" "$prefix/workspace-sandbox"

printf '%s\n' "$prefix/workspace-sandbox"
