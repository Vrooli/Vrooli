package gherkin

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExpandsOutlinesAndKeepsBackground(t *testing.T) {
	parsed, err := parse(`# a comment
Feature: Counter
  Background:
    Given a counter at 1

  Scenario: add once
    When I add 2
    Then the counter is 3

  Scenario Outline: add many
    When I add <n>
    Then the counter is <total>
    Examples:
      | n | total |
      | 1 | 2     |
      | 4 | 5     |
`)
	require.NoError(t, err)
	require.Equal(t, "Counter", parsed.name)
	require.Len(t, parsed.background, 1)
	require.Len(t, parsed.scenarios, 3)
	require.Equal(t, "When I add 4", parsed.scenarios[2].steps[0].keyword+" "+parsed.scenarios[2].steps[0].text)
	require.Equal(t, "the counter is 5", parsed.scenarios[2].steps[1].text)
}

func TestParseRejectsMalformedFeatures(t *testing.T) {
	for name, text := range map[string]string{
		"step outside scenario":   "Feature: x\n  Given a thing\n",
		"outline without rows":    "Feature: x\n  Scenario Outline: o\n    Given <a>\n    Examples:\n      | a |\n",
		"ragged example row":      "Feature: x\n  Scenario Outline: o\n    Given <a>\n    Examples:\n      | a | b |\n      | 1 |\n",
		"missing feature header":  "Scenario: s\n  Given a thing\n",
		"unrecognized line shape": "Feature: x\n  Scenario: s\n    Whenever something\n",
	} {
		_, err := parse(text)
		require.Error(t, err, name)
	}
}

type counter struct{ n int }

func TestRunExecutesStepsAgainstAFreshWorldPerScenario(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "counter.feature"), []byte(`Feature: Counter
  Scenario: first
    Given a counter at 10
    When I add 5
    Then the counter is 15

  Scenario: second starts fresh
    Then the counter is 0
`), 0o600))
	RunFeatures(t, dir, func(*testing.T) *counter { return &counter{} }, func(s *Suite[counter]) {
		s.Step(`a counter at (\d+)`, func(_ *testing.T, w *counter, a []string) { w.n, _ = strconv.Atoi(a[0]) })
		s.Step(`I add (\d+)`, func(_ *testing.T, w *counter, a []string) { v, _ := strconv.Atoi(a[0]); w.n += v })
		s.Step(`the counter is (\d+)`, func(t *testing.T, w *counter, a []string) {
			v, _ := strconv.Atoi(a[0])
			require.Equal(t, v, w.n)
		})
	})
}
