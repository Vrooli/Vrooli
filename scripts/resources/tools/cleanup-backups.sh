#!/usr/bin/env bash
set -euo pipefail

# Backup Cleanup Script for Vrooli Resources
# Safely archives and removes backup files

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR=$(dirname "$SCRIPT_DIR")
BACKUP_ARCHIVE_DIR="$HOME/.vrooli/resource-backups"

echo "🧹 Cleaning up backup files in resources directory..."
echo "   Archive location: $BACKUP_ARCHIVE_DIR"
echo

# Create archive directories
mkdir -p "$BACKUP_ARCHIVE_DIR/manual"
mkdir -p "$BACKUP_ARCHIVE_DIR/original"
mkdir -p "$BACKUP_ARCHIVE_DIR/auto"

cd "$RESOURCES_DIR"

# Count backup files
AUTO_COUNT=$(find . -name "*.backup.auto" -o -name "*.bats.backup.auto" 2>/dev/null | wc -l)
MANUAL_COUNT=$(find . -name "*.backup.manual" -o -name "*.bats.backup.manual" 2>/dev/null | wc -l)
ORIGINAL_COUNT=$(find . -name "*.original" -o -name "*.bats.original" 2>/dev/null | wc -l)

echo "📊 Found backup files:"
echo "   - Auto backups: $AUTO_COUNT"
echo "   - Manual backups: $MANUAL_COUNT"
echo "   - Original files: $ORIGINAL_COUNT"
echo

# Archive manual backups (might be important)
if [ "$MANUAL_COUNT" -gt 0 ]; then
    echo "📦 Archiving manual backups..."
    find . -name "*.backup.manual" -o -name "*.bats.backup.manual" | while read -r file; do
        # Create relative path in archive
        REL_PATH=$(dirname "$file" | sed 's|^\./||')
        mkdir -p "$BACKUP_ARCHIVE_DIR/manual/$REL_PATH"
        mv "$file" "$BACKUP_ARCHIVE_DIR/manual/$REL_PATH/"
        echo "   Archived: $file"
    done
fi

# Archive .original files
if [ "$ORIGINAL_COUNT" -gt 0 ]; then
    echo "📦 Archiving original files..."
    find . -name "*.original" -o -name "*.bats.original" | while read -r file; do
        # Create relative path in archive
        REL_PATH=$(dirname "$file" | sed 's|^\./||')
        mkdir -p "$BACKUP_ARCHIVE_DIR/original/$REL_PATH"
        mv "$file" "$BACKUP_ARCHIVE_DIR/original/$REL_PATH/"
        echo "   Archived: $file"
    done
fi

# Remove auto backups older than 7 days
echo "🗑️  Removing auto backups older than 7 days..."
OLD_AUTO_COUNT=$(find . -name "*.backup.auto" -o -name "*.bats.backup.auto" -mtime +7 2>/dev/null | wc -l)
if [ "$OLD_AUTO_COUNT" -gt 0 ]; then
    find . -name "*.backup.auto" -o -name "*.bats.backup.auto" -mtime +7 -exec rm {} \;
    echo "   Removed $OLD_AUTO_COUNT old auto backups"
fi

# Archive recent auto backups
RECENT_AUTO_COUNT=$(find . -name "*.backup.auto" -o -name "*.bats.backup.auto" 2>/dev/null | wc -l)
if [ "$RECENT_AUTO_COUNT" -gt 0 ]; then
    echo "📦 Archiving recent auto backups..."
    find . -name "*.backup.auto" -o -name "*.bats.backup.auto" | while read -r file; do
        # Create relative path in archive
        REL_PATH=$(dirname "$file" | sed 's|^\./||')
        mkdir -p "$BACKUP_ARCHIVE_DIR/auto/$REL_PATH"
        mv "$file" "$BACKUP_ARCHIVE_DIR/auto/$REL_PATH/"
        echo "   Archived: $file"
    done
fi

# Clean up agent-s2 timestamped backups (keep only 2 most recent)
if [ -d "agents/agent-s2/backups" ]; then
    echo "🗑️  Cleaning agent-s2 timestamped backups..."
    cd agents/agent-s2/backups
    BACKUP_DIRS=$(ls -t 2>/dev/null | tail -n +3)
    if [ -n "$BACKUP_DIRS" ]; then
        echo "$BACKUP_DIRS" | xargs -r rm -rf
        echo "   Kept 2 most recent, removed: $(echo "$BACKUP_DIRS" | wc -l) old backups"
    fi
    cd "$RESOURCES_DIR"
fi

# Remove empty directories
echo "🗑️  Removing empty directories..."
find . -type d -empty -delete 2>/dev/null || true

echo
echo "✅ Cleanup complete!"
echo "   Archives saved to: $BACKUP_ARCHIVE_DIR"
echo "   You can safely delete the archive directory if these backups are no longer needed."