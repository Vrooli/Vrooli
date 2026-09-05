package workflowruntime

import (
	"strings"
	"testing"
)

func TestRenderPromptRendersStructuredEvaluatorBindingInsideSkillEnvelope(t *testing.T) {
	const source = `<skills count="1">
  <skill id="evaluator"><![CDATA[
Assess the treatment result:
{{.treatment}}
]]></skill>
</skills>`

	got, err := RenderPrompt(source, map[string]any{"treatment": "{\n  \"quality\": \"good\",\n  \"token\": \"alpha\"\n}"})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if strings.Contains(got, "{{.treatment}}") {
		t.Fatalf("RenderPrompt() left evaluator binding unresolved: %q", got)
	}
	if !strings.Contains(got, `"token": "alpha"`) || !strings.Contains(got, `"quality": "good"`) {
		t.Fatalf("RenderPrompt() did not embed structured treatment result: %q", got)
	}
}
