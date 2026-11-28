#!/usr/bin/env bash
set -euo pipefail

: "${TEST_GENIE_TARGET_API_BASE:?Test Genie must provide the target API URL}"
: "${TEST_GENIE_ROUTING_LEASE_ID:?Test Genie routed lease is required}"
: "${TEST_GENIE_SCENARIO_DIR:?Test Genie must provide the scenario directory}"

repo_root="${TEST_GENIE_REPO_ROOT:?Test Genie must provide the repository root}"
cli_dir="$repo_root/scenarios/device-studio/cli"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
debug_file="/tmp/device-studio-e1-routed-seed-${TEST_GENIE_ROUTING_LEASE_ID}.log"
printf 'lease=%s target=%s\n' "$TEST_GENIE_ROUTING_LEASE_ID" "$TEST_GENIE_TARGET_API_BASE" >"$debug_file"
exec 2> >(tee -a "$debug_file" >&2)

(
  cd "$cli_dir"
  go build -o "$tmp_dir/device-studio" .
)

export DEVICE_STUDIO_TEST_MODE=1
cli=("$tmp_dir/device-studio" --api-base "$TEST_GENIE_TARGET_API_BASE")
create_json="$tmp_dir/create.json"
list_before_json="$tmp_dir/list-before.json"
get_before_json="$tmp_dir/get-before.json"
rename_json="$tmp_dir/rename.json"
list_after_json="$tmp_dir/list-after.json"
get_after_json="$tmp_dir/get-after.json"
conflict_output="$tmp_dir/conflict.txt"

"${cli[@]}" projects create --name "E1 routed project" --json >"$create_json"
project_id="$(jq -er '.project.id' "$create_json")"
initial_revision="$(jq -er '.project.revision' "$create_json")"

"${cli[@]}" projects list --json >"$list_before_json"
jq -e --arg id "$project_id" 'any(.projects[]; .id == $id)' "$list_before_json" >/dev/null
"${cli[@]}" projects get "$project_id" --json >"$get_before_json"
jq -e --arg id "$project_id" --arg rev "$initial_revision" '.project.id == $id and (.project.revision | tostring) == $rev' "$get_before_json" >/dev/null

"${cli[@]}" projects rename "$project_id" --name "E1 routed project renamed" --expected-revision "$initial_revision" --json >"$rename_json"
current_revision="$(jq -er '.project.revision' "$rename_json")"
if "${cli[@]}" projects rename "$project_id" --name "E1 stale attempted rename" --expected-revision "$initial_revision" --json >"$tmp_dir/stale.json" 2>"$conflict_output"; then
  echo "stale rename unexpectedly succeeded" >&2
  exit 1
fi
if ! grep -Fq "expected revision $initial_revision, current revision $current_revision" "$conflict_output"; then
  echo "stale rename failed without the typed current revision detail:" >&2
  cat "$conflict_output" >&2
  exit 1
fi

"${cli[@]}" projects get "$project_id" --json >"$get_after_json"
jq -e --arg id "$project_id" --arg name "E1 routed project renamed" --arg rev "$current_revision" '.project.id == $id and .project.name == $name and (.project.revision | tostring) == $rev' "$get_after_json" >/dev/null
"${cli[@]}" projects list --json >"$list_after_json"
jq -e --arg id "$project_id" --arg name "E1 routed project renamed" 'any(.projects[]; .id == $id and .name == $name)' "$list_after_json" >/dev/null

state_path="$TEST_GENIE_SCENARIO_DIR/coverage/runtime/seed-state.json"
mkdir -p "$(dirname "$state_path")"
jq -n \
  --arg project_id "$project_id" \
  --arg project_name "E1 routed project renamed" \
  --argjson initial_revision "$initial_revision" \
  --argjson current_revision "$current_revision" \
  '{e1RoutedProject:{id:$project_id,name:$project_name,initialRevision:$initial_revision,currentRevision:$current_revision,listedBefore:true,openedBefore:true,staleConflictRejected:true,reopenedAfterRename:true,listedAfter:true}}' \
  >"$state_path"

echo "E1 routed project journey passed through the direct CLI with test-mode header; state recorded at $state_path"
