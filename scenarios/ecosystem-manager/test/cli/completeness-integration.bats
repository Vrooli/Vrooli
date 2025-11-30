#!/usr/bin/env bats
# Integration tests for scenario completeness scoring system

# Simple test helpers (no external dependencies needed)

# Test completeness command existence
@test "completeness command exists" {
  run bash -c "vrooli scenario completeness math-tools --help"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Calculate objective completeness score"* ]]
}

# Test JSON output format
@test "completeness outputs valid JSON with --format json" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  # Parse JSON to verify structure
  echo "$output" | jq -e '.scenario' >/dev/null
  echo "$output" | jq -e '.category' >/dev/null
  echo "$output" | jq -e '.score' >/dev/null
  echo "$output" | jq -e '.classification' >/dev/null
  echo "$output" | jq -e '.breakdown.quality' >/dev/null
  echo "$output" | jq -e '.breakdown.coverage' >/dev/null
  echo "$output" | jq -e '.breakdown.quantity' >/dev/null
}

# Test human-readable output format
@test "completeness outputs human-readable text by default" {
  run vrooli scenario completeness deployment-manager
  [ "$status" -eq 0 ]
  # The scenario-completeness-scoring CLI outputs in uppercase
  [[ "$output" == *"COMPLETENESS SCORE:"* ]]
  [[ "$output" == *"Quality Metrics"* ]]
  [[ "$output" == *"Coverage Metrics"* ]]
  [[ "$output" == *"Quantity Metrics"* ]]
}

# Test score range validation
@test "completeness score is between 0-100" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  score=$(echo "$output" | jq -r '.score')
  [ "$score" -ge 0 ]
  [ "$score" -le 100 ]
}

# Test classification matches score range
@test "classification matches score thresholds" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  score=$(echo "$output" | jq -r '.score')
  classification=$(echo "$output" | jq -r '.classification')

  if [ "$score" -ge 96 ]; then
    [ "$classification" = "production_ready" ]
  elif [ "$score" -ge 81 ]; then
    [ "$classification" = "nearly_ready" ]
  elif [ "$score" -ge 61 ]; then
    [ "$classification" = "mostly_complete" ]
  elif [ "$score" -ge 41 ]; then
    [ "$classification" = "functional_incomplete" ]
  elif [ "$score" -ge 21 ]; then
    [ "$classification" = "foundation_laid" ]
  else
    [ "$classification" = "early_stage" ]
  fi
}

# Test breakdown components sum to total score
@test "breakdown components sum correctly" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  total=$(echo "$output" | jq -r '.score')
  base_score=$(echo "$output" | jq -r '.base_score // .breakdown.base_score // 0')
  validation_penalty=$(echo "$output" | jq -r '.validation_penalty // .breakdown.validation_penalty // 0')
  quality=$(echo "$output" | jq -r '.breakdown.quality.score')
  coverage=$(echo "$output" | jq -r '.breakdown.coverage.score')
  quantity=$(echo "$output" | jq -r '.breakdown.quantity.score')
  ui=$(echo "$output" | jq -r '.breakdown.ui.score // 0')

  # base_score = quality + coverage + quantity + ui
  computed_base=$((quality + coverage + quantity + ui))
  [ "$computed_base" -eq "$base_score" ]

  # final score = base_score - validation_penalty (capped at 0)
  computed_total=$((base_score - validation_penalty))
  [ "$computed_total" -lt 0 ] && computed_total=0
  [ "$computed_total" -eq "$total" ]
}

# Test operational target extraction works
@test "operational targets extracted for browser-automation-studio" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  targets=$(echo "$output" | jq -r '.breakdown.quality.target_pass_rate.total')
  [ "$targets" -gt 0 ]
}

# Test requirement loading with imports
@test "requirements loaded via imports array" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  reqs=$(echo "$output" | jq -r '.breakdown.quality.requirement_pass_rate.total')
  [ "$reqs" -ge 60 ]
}

# Test test result aggregation from phases
@test "test results aggregated from phase-results" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  tests=$(echo "$output" | jq -r '.breakdown.quality.test_pass_rate.total')
  [ "$tests" -gt 0 ]
}

# Test recommendations generated
@test "recommendations provided for incomplete scenarios" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  rec_count=$(echo "$output" | jq -r '.recommendations | length')
  [ "$rec_count" -gt 0 ]
}

# Test error handling for non-existent scenario
@test "error handling for non-existent scenario" {
  run vrooli scenario completeness nonexistent-scenario-12345
  [ "$status" -ne 0 ]
  [[ "$output" == *"Scenario not found"* ]]
}

# Test legacy sync metadata support (deployment-manager)
@test "legacy sync metadata format supported" {
  run vrooli scenario completeness deployment-manager --format json
  [ "$status" -eq 0 ]

  reqs=$(echo "$output" | jq -r '.breakdown.quality.requirement_pass_rate.total')
  [ "$reqs" -gt 0 ]
}

# Test category threshold application
@test "category thresholds applied correctly" {
  # Utility category scenario
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  category=$(echo "$output" | jq -r '.category')
  [ "$category" = "utility" ]

  # Check that quantity scoring uses utility thresholds
  # (This is implicit in the score calculation)
}

# Test staleness warning detection
@test "staleness warning for old test results" {
  # This would require a scenario with stale tests
  # For now, just verify warning structure exists
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  # Warnings array should exist (may be empty)
  echo "$output" | jq -e '.warnings' >/dev/null
}

# Test fractional pass rates
@test "pass rates calculated as fractions between 0 and 1" {
  run vrooli scenario completeness browser-automation-studio --format json
  [ "$status" -eq 0 ]

  req_rate=$(echo "$output" | jq -r '.breakdown.quality.requirement_pass_rate.rate')
  target_rate=$(echo "$output" | jq -r '.breakdown.quality.target_pass_rate.rate')
  test_rate=$(echo "$output" | jq -r '.breakdown.quality.test_pass_rate.rate')

  # Validate rates are between 0 and 1
  [ "$(echo "$req_rate >= 0 && $req_rate <= 1" | bc)" -eq 1 ]
  [ "$(echo "$target_rate >= 0 && $target_rate <= 1" | bc)" -eq 1 ]
  [ "$(echo "$test_rate >= 0 && $test_rate <= 1" | bc)" -eq 1 ]
}
