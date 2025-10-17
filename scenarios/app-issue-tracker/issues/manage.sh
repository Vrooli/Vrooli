#!/usr/bin/env bash
# Issue Tracker helper utilities for folder-based storage bundles

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUES_DIR="$SCRIPT_DIR"

STATUS_FOLDERS=(open investigating in-progress fixed closed failed)

# Colours
CYAN='\033[1;36m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
RED='\033[1;31m'
NC='\033[0m'

banner() { echo -e "${CYAN}$1${NC}"; }
info()   { echo -e "${CYAN}ℹ${NC} $1"; }
warn()   { echo -e "${YELLOW}⚠${NC} $1"; }
error()  { echo -e "${RED}✖${NC} $1"; exit 1; }

require_dir() {
  local dir="$1"
  [[ -d "$ISSUES_DIR/$dir" ]] || error "Unknown status '$dir'."
}

count_bundles() {
  local dir="$1"
  find "$ISSUES_DIR/$dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' '
}

extract_field() {
  local field="$1"
  local file="$2"
  awk -F': ' -v key="$field" 'tolower($1)==tolower(key) {gsub(/"/,"",$2); print $2; exit}' "$file"
}

status() {
  banner "🐛 Issue Bundle Status"
  local total=0
  for folder in "${STATUS_FOLDERS[@]}"; do
    local count
    count=$(count_bundles "$folder")
    printf "%-14s %s\n" "$folder" "$count"
    total=$((total + count))
  done
  printf '\nTotal bundles: %s\n' "$total"
}

list_state() {
  local folder="${1:-open}"
  require_dir "$folder"
  banner "📂 ${folder} issues"

  local found=false
  for issue_dir in "$ISSUES_DIR/$folder"/*/; do
    [[ -d "$issue_dir" ]] || continue
    local metadata="${issue_dir%/}/metadata.yaml"
    [[ -f "$metadata" ]] || continue

    local id title priority updated
    id=$(extract_field id "$metadata")
    title=$(extract_field title "$metadata")
    priority=$(extract_field priority "$metadata")
    updated=$(grep -m1 'updated_at:' "$metadata" | awk -F': ' '{gsub(/"/,"",$2); print $2}')

    printf "%s%-40s%s\n" "${GREEN}" "${id:-$(basename "$issue_dir")}" "${NC}"
    printf "  • Title: %s\n" "${title:-<no title>}"
    printf "  • Priority: %s\n" "${priority:-unknown}"
    [[ -n "$updated" ]] && printf "  • Updated: %s\n" "$updated"
    printf "  • Path: %s\n\n" "${issue_dir#$ISSUES_DIR/}"
    found=true
  done

  $found || info "No issues found in $folder/"
}

show_issue() {
  local issue_id="$1"
  [[ -n "$issue_id" ]] || error "Usage: $0 show <issue-id>"

  for folder in "${STATUS_FOLDERS[@]}"; do
    local candidate="$ISSUES_DIR/$folder/$issue_id"
    if [[ -d "$candidate" ]]; then
      banner "🧾 $issue_id ($folder)"
      cat "$candidate/metadata.yaml"
      echo
      if [[ -d "$candidate/artifacts" ]]; then
        banner "📎 Artifacts"
        ls -1 "$candidate/artifacts"
      else
        info "No artifacts captured"
      fi
      return
    fi
  done

  error "Issue '$issue_id' not found"
}

add_issue() {
  warn "Interactive bundle creation is now handled by the main CLI/API."
  echo "Run: app-issue-tracker create --title \"...\" --priority medium" >&2
}

usage() {
  cat <<USAGE
Usage: $(basename "$0") <command> [args]

Commands:
  status                Show bundle counts per status
  list [status]         List issues (default: open)
  show <issue-id>       Display metadata + artifacts for a bundle
  add                   Print instructions for creating an issue via API
  help                  Show this message
USAGE
}

case "${1:-help}" in
  status) status ;;
  list)   list_state "${2:-open}" ;;
  show)   shift; show_issue "${1:-}" ;;
  add)    add_issue ;;
  help|--help|-h) usage ;;
  *) usage; exit 1 ;;
esac
