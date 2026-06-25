#!/usr/bin/env bash
# Oracle for the research fixture. Runs with cwd = the diff-applied copy of
# target/. Passes iff FOUND.txt holds the token hidden under data/.
set -u

if [ ! -f FOUND.txt ]; then
  echo "FOUND.txt was not created" >&2
  exit 1
fi

got="$(tr -d '[:space:]' < FOUND.txt)"
if [ "$got" != "z9x8c7v6" ]; then
  echo "FOUND.txt = '$got', want 'z9x8c7v6'" >&2
  exit 1
fi
exit 0
