#!/usr/bin/env bats

setup() {
  export PATH="$HOME/.vrooli/bin:$PATH"
}

@test "CLI help displays usage" {
  run no-spoilers-book-talk --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"No Spoilers Book Talk CLI"* ]]
}

@test "CLI status command executes" {
  run no-spoilers-book-talk status
  [ "$status" -eq 0 ]
  [[ "$output" == *"Scenario"* || "$output" == *"No Spoilers Book Talk"* ]]
}
