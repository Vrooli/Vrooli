#!/usr/bin/env sh
set -eu

prefix=${HELLO_PLUGIN_PREFIX:-"${XDG_BIN_HOME:-$HOME/.local/bin}"}
mkdir -p "$prefix"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
target="$prefix/hello-plugin"
tmp="$target.tmp.$$"
cp "$script_dir/hello-plugin" "$tmp"
chmod 0755 "$tmp"
mv "$tmp" "$target"
printf '%s\n' "$target"
