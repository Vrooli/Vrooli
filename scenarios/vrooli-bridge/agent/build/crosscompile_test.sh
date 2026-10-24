#!/usr/bin/env bash
# crosscompile_test.sh — [REQ:BRG-P0-007] integration gate.
#
# The node-agent is the single cross-built client installed on each trusted
# node. It MUST build CGO_ENABLED=0 for the full linux/darwin/windows ×
# amd64/arm64 matrix so a node can run any supported OS from day one (OT-P0-007).
# This script compiles every target into a temp dir and fails if ANY target does
# not build, proving the one agent codebase carries no Linux-only assumptions and
# stays statically linkable across the matrix.
#
# It is the executable form of `make matrix` (same PLATFORMS list) intended for
# CI and the requirements-traceability `integration` validation; it builds to a
# throwaway directory and leaves no artifacts behind.
set -euo pipefail

# Resolve the agent module root (this script lives in agent/build/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# The authoritative cross-compile matrix. Keep in lockstep with agent/Makefile's
# PLATFORMS list.
PLATFORMS=(
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
  windows/amd64
  windows/arm64
)

OUT_DIR="$(mktemp -d)"
trap 'rm -rf "${OUT_DIR}"' EXIT

echo "cross-compiling vrooli-bridge-agent (CGO_ENABLED=0) for ${#PLATFORMS[@]} targets"
fail=0
for p in "${PLATFORMS[@]}"; do
  os="${p%/*}"
  arch="${p#*/}"
  ext=""
  [ "${os}" = "windows" ] && ext=".exe"
  out="${OUT_DIR}/vrooli-bridge-agent-${os}-${arch}${ext}"
  # Build from the nested agent module. Passing its absolute directory as a
  # package path makes Go resolve it through the repository's workspace module
  # instead of the agent's own go.mod when GOWORK is disabled.
  if (cd "${AGENT_DIR}" && CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" GOWORK=off \
      go build -trimpath -o "${out}" .); then
    echo "  ok    ${p}"
  else
    echo "  FAIL  ${p}"
    fail=1
  fi
done

if [ "${fail}" -ne 0 ]; then
  echo "cross-compile matrix FAILED" >&2
  exit 1
fi
echo "cross-compile matrix OK (${#PLATFORMS[@]}/${#PLATFORMS[@]} targets)"
