#!/usr/bin/env bash
# Oracle for the bugfix fixture. Runs with cwd = the diff-applied copy of target/.
set -u

assert_eq() {
  local got want
  got="$(bash parity.sh "$1" 2>/dev/null)"
  want="$2"
  if [ "$got" != "$want" ]; then
    echo "parity.sh $1 => '$got', want '$want'" >&2
    exit 1
  fi
}

assert_eq 3 odd
assert_eq 4 even
assert_eq 0 even
assert_eq -3 odd
assert_eq -4 even
exit 0
