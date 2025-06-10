#!/usr/bin/env bash
###############################################################################
# print_tree_contents.sh
#
# Recursively echoes the *text* contents of every regular file beneath a given
# directory.  Each file is printed in the form
#
#   <relative/path/to/file>:
#   """
#   <file contents>
#   """
#
# Binary files (as detected by the `file` command) are skipped to avoid
# polluting the output with raw bytes.
#
# USAGE
#   ./print_tree_contents.sh [ROOT_DIR]
#
#   ROOT_DIR  Optional. Directory to traverse.  Defaults to the current 
#             working directory.
#
# EXAMPLES
#   # Print everything under the current directory
#   ./print_tree_contents.sh
#
#   # Print everything under ./src
#   ./print_tree_contents.sh ./src
#
###############################################################################

set -euo pipefail

ROOT_DIR="${1:-.}"

# Abort early if the path doesn’t exist or isn’t a directory.
if [[ ! -d "$ROOT_DIR" ]]; then
  echo "Error: '$ROOT_DIR' is not a directory." >&2
  exit 1
fi

# Traverse all regular files.  Use NUL delimiters so we cope with spaces, tabs,
# newlines, etc. in filenames.
find "$ROOT_DIR" -type f -print0 |
while IFS= read -r -d '' FILE; do
  # Skip binary files (heuristic: `file --mime-encoding` ≠ "binary").
  if [[ "$(file --brief --mime-encoding "$FILE")" == "binary" ]]; then
    continue
  fi

  # Convert absolute FILE to a path relative to ROOT_DIR for nicer output.
  REL_PATH="${FILE#$ROOT_DIR/}"

  printf '<%s>:\n"""\n' "$REL_PATH"
  cat "$FILE"
  printf '\n"""\n\n'
done
