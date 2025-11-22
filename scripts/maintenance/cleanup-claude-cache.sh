#!/bin/bash
################################################################################
# Claude Code Cache Cleanup Script
#
# Purpose: Clean up old Claude Code cache files that cause ENOSPC errors
# Root Cause: Claude Code watches ~/.claude/ directory with 10,000+ files
#
# When to run: If you see "ENOSPC: System limit for number of file watchers"
# at Claude Code startup
################################################################################

set -euo pipefail

CLAUDE_DIR="${HOME}/.claude"
DRY_RUN="${DRY_RUN:-false}"

echo "🧹 Claude Code Cache Cleanup"
echo "======================================================"
echo ""

if [[ ! -d "$CLAUDE_DIR" ]]; then
    echo "❌ Claude directory not found: $CLAUDE_DIR"
    exit 1
fi

# Count current files
echo "📊 Current state:"
echo "  todos:           $(find "$CLAUDE_DIR/todos" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/todos" 2>/dev/null | cut -f1))"
echo "  file-history:    $(find "$CLAUDE_DIR/file-history" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/file-history" 2>/dev/null | cut -f1))"
echo "  shell-snapshots: $(find "$CLAUDE_DIR/shell-snapshots" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/shell-snapshots" 2>/dev/null | cut -f1))"
echo ""

# Check if Claude is running
if pgrep -f "^claude$" >/dev/null; then
    echo "⚠️  Warning: Claude Code sessions are currently running"
    echo "   It's safe to continue, but you may want to close them first."
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 0
    fi
fi

# Cleanup function
cleanup_old_files() {
    local dir="$1"
    local days="$2"
    local description="$3"

    if [[ ! -d "$dir" ]]; then
        echo "⏭️  Skipping $description (directory doesn't exist)"
        return 0
    fi

    local count=$(find "$dir" -type f -mtime "+${days}" 2>/dev/null | wc -l)

    if [[ $count -eq 0 ]]; then
        echo "✅ No old files to clean in $description"
        return 0
    fi

    echo "🗑️  Found $count files older than $days days in $description"

    if [[ "$DRY_RUN" == "true" ]]; then
        echo "   [DRY RUN] Would delete these files"
        return 0
    fi

    find "$dir" -type f -mtime "+${days}" -delete 2>/dev/null
    echo "   ✅ Deleted $count files"
}

echo "🧹 Cleaning up old cache files..."
echo ""

# Clean up old todos (keep last 7 days)
cleanup_old_files "$CLAUDE_DIR/todos" 7 "todos"

# Clean up old file history (keep last 30 days)
cleanup_old_files "$CLAUDE_DIR/file-history" 30 "file-history"

# Clean up old shell snapshots (keep last 7 days)
cleanup_old_files "$CLAUDE_DIR/shell-snapshots" 7 "shell-snapshots"

# Clean up empty directories
echo ""
echo "🧹 Cleaning up empty directories..."
find "$CLAUDE_DIR" -type d -empty -delete 2>/dev/null || true

echo ""
echo "📊 After cleanup:"
echo "  todos:           $(find "$CLAUDE_DIR/todos" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/todos" 2>/dev/null | cut -f1))"
echo "  file-history:    $(find "$CLAUDE_DIR/file-history" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/file-history" 2>/dev/null | cut -f1))"
echo "  shell-snapshots: $(find "$CLAUDE_DIR/shell-snapshots" -type f 2>/dev/null | wc -l) files ($(du -sh "$CLAUDE_DIR/shell-snapshots" 2>/dev/null | cut -f1))"

echo ""
echo "✅ Cleanup complete!"
echo ""
echo "💡 To prevent this issue in the future, consider:"
echo "   1. Running this script periodically (weekly)"
echo "   2. Adding it to a cron job: 0 0 * * 0 $0"
echo "   3. Reporting to Claude Code team that file watching should exclude cache dirs"
