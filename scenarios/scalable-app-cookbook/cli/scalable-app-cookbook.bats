#!/usr/bin/env bats

setup() {
  run command -v scalable-app-cookbook
  [ "$status" -eq 0 ]
}

@test "scalable-app-cookbook help displays usage" {
  run scalable-app-cookbook help
  [ "$status" -eq 0 ]
  [[ "$output" == *"USAGE:"* ]]
}
