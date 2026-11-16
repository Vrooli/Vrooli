package conversations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestReadTemplateDataRequiresAJSONObject(t *testing.T) {
	path := writeFile(t, t.TempDir(), "data.json", `{"count":3,"enabled":true,"series":[1,2]}`)
	got, err := readTemplateData(path, nil)
	require.NoError(t, err)
	require.Equal(t, float64(3), got["count"])
	require.Equal(t, true, got["enabled"])
	require.Equal(t, []any{float64(1), float64(2)}, got["series"])

	null := writeFile(t, t.TempDir(), "null.json", `null`)
	_, err = readTemplateData(null, nil)
	require.ErrorContains(t, err, "not null")

	array := writeFile(t, t.TempDir(), "array.json", `[]`)
	_, err = readTemplateData(array, nil)
	require.ErrorContains(t, err, "JSON object")

	broken := writeFile(t, t.TempDir(), "broken.json", `{"count":`)
	_, err = readTemplateData(broken, nil)
	require.ErrorContains(t, err, "must contain a JSON object")
}

func TestAskSendsKitTemplateAndJSONData(t *testing.T) {
	path := writeFile(t, t.TempDir(), "data.json", `{"pace":"thursday","slots":4}`)
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", "--question", "Use this schedule?", "--yes-no", "--subject-key", "pace", "--template", "daily", "--data-file", path)
	require.NoError(t, err)
	require.Len(t, stub.asks, 1)
	req := stub.asks[0]
	require.Equal(t, "daily", req.GetTemplateName())
	require.Equal(t, map[string]any{"pace": "thursday", "slots": float64(4)}, req.GetTemplateData().AsMap())
	require.Equal(t, "decision", req.GetKind())

	stdin := &hubStub{}
	svc := newTestService(t, stdin)
	svc.stdin = strings.NewReader(`{"enabled":true}`)
	_, _, err = runCommand(t, svc, "ask", "--question", "Review this update?", "--kind", "status", "--template", "daily", "--data-file", "-")
	require.NoError(t, err)
	require.Equal(t, "status", stdin.asks[0].GetKind())
	require.Empty(t, stdin.asks[0].GetOptions(), "a status update has no answer menu")
	require.Equal(t, map[string]any{"enabled": true}, stdin.asks[0].GetTemplateData().AsMap())
}

func TestTemplateFlagsMustBePairedAndStatusCannotHaveDecisionOptions(t *testing.T) {
	svc := newTestService(t, &hubStub{})
	_, _, err := runCommand(t, svc, "ask", "--question", "Review?", "--yes-no", "--template", "daily")
	require.ErrorContains(t, err, "must be given together")
	_, _, err = runCommand(t, svc, "ask", "--question", "Review?", "--kind", "status", "--yes-no")
	require.ErrorContains(t, err, "cannot be combined")
	_, _, err = runCommand(t, svc, "ask", "--question", "Review?", "--kind", "surprise", "--yes-no")
	require.ErrorContains(t, err, "must be decision or status")
}

func TestReadTemplateDataConvertsToHubStruct(t *testing.T) {
	input, err := readTemplateData("-", strings.NewReader(`{"items":["a","b"]}`))
	require.NoError(t, err)
	got, err := structpb.NewStruct(input)
	require.NoError(t, err)
	encoded, err := json.Marshal(got.AsMap())
	require.NoError(t, err)
	require.JSONEq(t, `{"items":["a","b"]}`, string(encoded))
}

func TestReadTemplateDataSupportsStdin(t *testing.T) {
	got, err := readTemplateData("-", strings.NewReader(`{"pace":"thursday"}`))
	require.NoError(t, err)
	require.Equal(t, "thursday", got["pace"])
}
