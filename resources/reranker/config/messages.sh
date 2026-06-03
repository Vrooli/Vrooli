#!/usr/bin/env bash
# Reranker resource log helpers. Sourced by test/integration-test.sh for
# consistent, colorized status output. Kept intentionally small — the lifecycle
# is driven by resource.json, not bash.

msg::info()  { printf '\033[0;34m[reranker]\033[0m %s\n' "$*"; }
msg::ok()    { printf '\033[0;32m[reranker] ✓\033[0m %s\n' "$*"; }
msg::warn()  { printf '\033[0;33m[reranker] !\033[0m %s\n' "$*" >&2; }
msg::error() { printf '\033[0;31m[reranker] ✗\033[0m %s\n' "$*" >&2; }
