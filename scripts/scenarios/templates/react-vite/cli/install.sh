#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${VROOLI_BIN_DIR:-${HOME}/.vrooli/bin}"
CLI_NAME="{{SCENARIO_ID}}"
TARGET="${INSTALL_DIR}/${CLI_NAME}"

mkdir -p "$INSTALL_DIR"
cp "$SCRIPT_DIR/$CLI_NAME" "$TARGET"
chmod +x "$TARGET"

echo "Installed ${CLI_NAME} to ${INSTALL_DIR}"

append_if_missing() {
  local file="$1"
  local line="$2"
  if [ -f "$file" ]; then
    grep -Fqs "$line" "$file" || echo "$line" >>"$file"
  else
    echo "$line" >>"$file"
  fi
}

PATH_SNIPPET="export PATH=\"\$PATH:${INSTALL_DIR}\""

case "${SHELL:-}" in
  *zsh*) append_if_missing "$HOME/.zshrc" "${PATH_SNIPPET}" ;;
  *fish)
    mkdir -p "$HOME/.config/fish"
    append_if_missing "$HOME/.config/fish/config.fish" "set -gx PATH ${INSTALL_DIR} \$PATH"
    ;;
  *) append_if_missing "$HOME/.bashrc" "${PATH_SNIPPET}" ;;
esac

echo "Add ${INSTALL_DIR} to your PATH if the command is not yet available."
