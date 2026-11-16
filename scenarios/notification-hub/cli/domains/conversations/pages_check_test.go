package conversations

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// runPageCheck runs `pages check args...` through the real manifest entry.
func runPageCheck(t *testing.T, svc *service, args ...string) (string, error) {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := cliapp.LoadFromManifest(manifest, "pages", map[string]func(cliapp.RunContext) error{
		"check": func(ctx cliapp.RunContext) error { return checkPage(ctx, svc.client) },
		"shoot": func(cliapp.RunContext) error { return nil },
	})
	require.NoError(t, err)
	for _, cmd := range group.Subcommands {
		if cmd.Name == "check" {
			var out, errOut bytes.Buffer
			ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &out, &errOut)
			require.NoError(t, err)
			runErr := cmd.RunCtx(ctx)
			return out.String(), runErr
		}
	}
	t.Fatal("no pages check command")
	return "", nil
}

// `pages check` sends the page and its cover to the same check as a dry run,
// with the page's content, and fails when the Hub would refuse the ask.
func TestPageCheckRunsTheDryRunCheckOnAPageAndItsCover(t *testing.T) {
	dir := t.TempDir()
	page := writeFile(t, dir, "page.html", "<!doctype html><title>Cleanup</title><h1>Delete them?</h1>")
	cover := writeFile(t, dir, "request.json", `{"sender":"Writer","question":"Delete 104 test copies?",
		"options":[{"key":"delete","label":"Delete","effect":"gone"},{"key":"keep","label":"Keep"}],
		"recommended":"delete","recommendation_reason":"Nothing uses them."}`)
	stub := &hubStub{checkResp: &v1.CheckAskResponse{Warnings: []*v1.PacketWarning{
		{Code: "explainer_restates", Field: "attachments[0]", Message: "The page repeats the cover.", Severity: "blocking"},
	}}}
	out, err := runPageCheck(t, newTestService(t, stub), page, "--cover-file", cover)
	require.Error(t, err)
	require.Contains(t, out, "would refuse this decision")
	require.Contains(t, out, "The page repeats the cover. (explainer_restates)")

	require.Len(t, stub.checks, 1)
	sent := stub.checks[0]
	require.Equal(t, "Delete 104 test copies?", sent.GetAsk().GetQuestion())
	require.Equal(t, "Nothing uses them.", sent.GetAsk().GetRecommendationReason())
	require.Len(t, sent.GetAsk().GetAttachments(), 1)
	explainer := sent.GetAsk().GetAttachments()[0]
	require.Equal(t, v1.AskAttachment_ROLE_EXPLAINER, explainer.GetRole())
	require.Contains(t, string(sent.GetFileContents()[explainer.GetSha256()]), "Delete them?", "the Hub reads the page itself")
}

// Without a cover only the page is judged: whether it restates a cover cannot
// be told, so that finding is not reported, and a clean page passes.
func TestPageCheckWithoutACoverJudgesThePageAlone(t *testing.T) {
	dir := t.TempDir()
	page := writeFile(t, dir, "page.html", "<!doctype html><title>Cleanup</title><p>Text</p>")
	stub := &hubStub{checkResp: &v1.CheckAskResponse{Warnings: []*v1.PacketWarning{
		{Code: "explainer_restates", Field: "attachments[0]", Message: "Repeats.", Severity: "blocking"},
		{Code: "missing_big_picture", Field: "big_picture", Message: "No big picture.", Severity: "advice"},
		{Code: "explainer_no_viewport", Field: "attachments[0]", Message: "No viewport.", Severity: "advice"},
	}}}
	out, err := runPageCheck(t, newTestService(t, stub), page)
	require.NoError(t, err)
	require.Contains(t, out, "Checked the page alone")
	require.Contains(t, out, "No viewport.")
	require.NotContains(t, out, "Repeats.")
	require.NotContains(t, out, "No big picture.")
	require.Equal(t, "Cleanup", stub.checks[0].GetAsk().GetQuestion())
}
