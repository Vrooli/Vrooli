package inference

import (
	"testing"

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

func TestValidateJSONRejectsMalformedSchemaWithoutPanicking(t *testing.T) {
	// ValidateJSON is exported, so a schema that never passed SchemaGate.Parse
	// can reach it. It must return an error rather than panic.
	require.NotPanics(t, func() {
		err := ValidateJSON(map[string]any{"required": []any{42}}, []byte(`{"a":1}`))
		require.Error(t, err)
	})
	require.NotPanics(t, func() {
		err := ValidateJSON(map[string]any{"properties": map[string]any{"a": "not-a-schema"}}, []byte(`{"a":1}`))
		require.Error(t, err)
	})
}
