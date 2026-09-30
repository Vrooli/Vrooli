#!/usr/bin/env bash
# Oracle for the add-feature fixture. Runs with cwd = the diff-applied copy of
# target/. Exit 0 = solved. Kept OUTSIDE target/ so the agent cannot read it.
set -u

assert_eq() {
  local got want
  got="$(bash sum.sh "$1" "$2" 2>/dev/null)"
  want="$3"
  if [ "$got" != "$want" ]; then
    echo "sum.sh $1 $2 => '$got', want '$want'" >&2
    exit 1
  fi
}

assert_eq 2 3 5
assert_eq 10 -4 6
assert_eq 0 0 0
assert_eq 100 250 350
exit 0
