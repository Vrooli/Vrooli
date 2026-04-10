#!/usr/bin/env bats

setup() {
  TEST_DIR="$(mktemp -d)"
  export TEST_DIR
  export CHECKER="${BATS_TEST_DIRNAME}/ui-bundle-check.sh"

  mkdir -p "$TEST_DIR/ui/dist" "$TEST_DIR/ui/src" "$TEST_DIR/packages/api-base/src"
  printf '{}' > "$TEST_DIR/ui/src/app.json"
  printf '{}' > "$TEST_DIR/packages/api-base/src/index.json"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "returns setup-needed when bundle is missing" {
  run env APP_ROOT="$TEST_DIR" "$CHECKER" '{"bundle_path":"ui/dist/index.html","source_dir":"ui/src"}'
  [ "$status" -eq 0 ]
}

@test "returns setup-needed when file dependency is newer than bundle" {
  cat > "$TEST_DIR/ui/package.json" <<'EOF'
{
  "dependencies": {
    "@vrooli/api-base": "file:../packages/api-base"
  }
}
EOF
  # Ensure package.json/source are older than bundle so dependency drift drives result.
  touch -d '2026-02-19 00:00:00' "$TEST_DIR/ui/package.json" "$TEST_DIR/ui/src/app.json"
  touch -d '2026-02-19 00:10:00' "$TEST_DIR/ui/dist/index.html"
  touch -d '2026-02-19 00:20:00' "$TEST_DIR/packages/api-base/src/index.json"

  run env APP_ROOT="$TEST_DIR" "$CHECKER" '{"bundle_path":"ui/dist/index.html","source_dir":"ui/src"}'
  [ "$status" -eq 0 ]
}

@test "returns current when bundle is newer than file dependency" {
  cat > "$TEST_DIR/ui/package.json" <<'EOF'
{
  "dependencies": {
    "@vrooli/api-base": "file:../packages/api-base"
  }
}
EOF
  touch -d '2026-02-19 00:00:00' "$TEST_DIR/ui/package.json" "$TEST_DIR/ui/src/app.json" "$TEST_DIR/packages/api-base/src/index.json"
  touch -d '2026-02-19 00:10:00' "$TEST_DIR/ui/dist/index.html"

  run env APP_ROOT="$TEST_DIR" "$CHECKER" '{"bundle_path":"ui/dist/index.html","source_dir":"ui/src"}'
  [ "$status" -eq 1 ]
}

@test "can disable file dependency watch" {
  cat > "$TEST_DIR/ui/package.json" <<'EOF'
{
  "dependencies": {
    "@vrooli/api-base": "file:../packages/api-base"
  }
}
EOF
  touch -d '2026-02-19 00:00:00' "$TEST_DIR/ui/package.json" "$TEST_DIR/ui/src/app.json"
  touch -d '2026-02-19 00:10:00' "$TEST_DIR/ui/dist/index.html"
  touch -d '2026-02-19 00:20:00' "$TEST_DIR/packages/api-base/src/index.json"

  run env APP_ROOT="$TEST_DIR" "$CHECKER" '{"bundle_path":"ui/dist/index.html","source_dir":"ui/src","watch_file_dependencies":false}'
  [ "$status" -eq 1 ]
}
