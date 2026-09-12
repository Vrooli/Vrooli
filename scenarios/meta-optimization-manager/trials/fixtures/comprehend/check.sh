#!/usr/bin/env bash
# Oracle for the comprehend fixture. Runs with cwd = the diff-applied copy of
# target/. Passes iff ANSWER.txt names the single function defined in lib.sh.
set -u

if [ ! -f ANSWER.txt ]; then
  echo "ANSWER.txt was not created" >&2
  exit 1
fi

got="$(tr -d '[:space:]' < ANSWER.txt)"
if [ "$got" != "compute_checksum" ]; then
  echo "ANSWER.txt = '$got', want 'compute_checksum'" >&2
  exit 1
fi
exit 0
