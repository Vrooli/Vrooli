package inference

import (
	"testing"

	"ai-gateway/internal/schemagate"

	"github.com/stretchr/testify/require"
)

func TestExtractJSONValueRecoversFromCommonProviderFormatting(t *testing.T) {
	for name, testCase := range map[string]struct{ raw, want string }{
		"bare value": {
			raw:  `{"label":"infra"}`,
			want: `{"label":"infra"}`,
		},
		"fenced with language tag": {
			raw:  "```json\n{\"label\":\"infra\"}\n```",
			want: `{"label":"infra"}`,
		},
		"fenced without language tag": {
			raw:  "```\n{\"label\":\"infra\"}\n```",
			want: `{"label":"infra"}`,
		},
		"fenced on one line": {
			raw:  "```json {\"label\":\"infra\"}```",
			want: `{"label":"infra"}`,
		},
		"reasoning block before value": {
			raw:  "<think>The port suggests a database.</think>\n{\"label\":\"infra\"}",
			want: `{"label":"infra"}`,
		},
		"prose preamble and trailer": {
			raw:  "Here is the result:\n{\"label\":\"infra\"}\nLet me know if you need more.",
			want: `{"label":"infra"}`,
		},
		"array value": {
			raw:  "Result: [1, 2, 3] done",
			want: `[1, 2, 3]`,
		},
		"braces inside a string literal": {
			raw:  `Answer: {"label":"a}b","note":"{"}`,
			want: `{"label":"a}b","note":"{"}`,
		},
		"escaped quote inside a string literal": {
			raw:  `{"label":"say \"hi\""}`,
			want: `{"label":"say \"hi\""}`,
		},
		"nested objects": {
			raw:  "text {\"outer\":{\"inner\":[{\"k\":1}]}} tail",
			want: `{"outer":{"inner":[{"k":1}]}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testCase.want, ExtractJSONValue(testCase.raw))
		})
	}
}

// Extraction must not invent structure. When there is no JSON value the
// original text is returned so local validation reports the real content.
func TestExtractJSONValueLeavesNonJSONTextForTheValidatorToReject(t *testing.T) {
	require.Equal(t, "I could not classify this.", ExtractJSONValue("  I could not classify this.  "))
	require.Equal(t, "{unterminated", ExtractJSONValue("{unterminated"))
}

func TestBalancedJSONScannerSeparatesStringAndStructuralState(t *testing.T) {
	scanner := balancedJSONScanner{openers: make([]rune, 0, 8)}
	input := `{"text":"escaped \" } ] still text", "nested":[{}]}`
	for index, char := range input {
		complete, invalid := scanner.consume(char)
		require.False(t, invalid, "character %q at byte %d", char, index)
		require.Equal(t, index == len(input)-1, complete, "character %q at byte %d", char, index)
	}
	require.Empty(t, scanner.openers)
	require.False(t, scanner.inString)
	require.False(t, scanner.escaped)
}

func TestBalancedJSONScannerRejectsMismatchedAndUnmatchedClosers(t *testing.T) {
	for name, input := range map[string]string{"mismatched": "{]", "unmatched": "}"} {
		t.Run(name, func(t *testing.T) {
			scanner := balancedJSONScanner{openers: make([]rune, 0, 8)}
			var invalid bool
			for _, char := range input {
				_, invalid = scanner.consume(char)
				if invalid {
					break
				}
			}
			require.True(t, invalid)
		})
	}
}

func TestValidateJSONRejectsMalformedSchemaWithoutPanicking(t *testing.T) {
	// schemagate.ValidateJSON is exported, so a schema that never passed schemagate.SchemaGate.Parse
	// can reach it. It must return an error rather than panic.
	require.NotPanics(t, func() {
		err := schemagate.ValidateJSON(map[string]any{"required": []any{42}}, []byte(`{"a":1}`))
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		err := schemagate.ValidateJSON(map[string]any{"properties": map[string]any{"a": "not-a-schema"}}, []byte(`{"a":1}`))
		require.Error(t, err)
	})
}
