#!/usr/bin/env bash
# Test helpers for resource-playwright (stub)

playwright::test::smoke() {
  "${BASH_SOURCE%/*}/../cli.sh" test
}
