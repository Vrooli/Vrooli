package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The pages in testdata/page-quality are real decisions Codex Luna wrote for
// six real cases in the decision-quality eval
// (~/.vrooli/research/notification-hub-decision-eval): "baseline" with the old
// skill, "after" with revision 10, "after-2" with revision 11. The operator's
// complaint was about the baseline kind: pages that tell the decision again in
// prose cards. Each baseline page must draw a page finding, and no after-2
// page may.
var pageQualityCodes = map[string]bool{
	WarnExplainerRestates: true, WarnPageNoPicture: true, WarnPagePictureUncropped: true,
	WarnCommandHidden: true, WarnPageUnreadable: true,
}

func pageQualityFindings(t *testing.T, round string) map[string][]string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join("testdata", "page-quality", round, "*"))
	require.NoError(t, err)
	require.Len(t, dirs, 6, round)
	out := map[string][]string{}
	for _, dir := range dirs {
		var cover struct {
			Question, BigPicture, Recommend, Reason, TechnicalDetails string
			Options                                                   []AskOption
		}
		raw, err := os.ReadFile(filepath.Join(dir, "cover.json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(strings.NewReplacer(`"big_picture"`, `"BigPicture"`, `"technical_details"`, `"TechnicalDetails"`).Replace(string(raw))), &cover))
		page, err := os.ReadFile(filepath.Join(dir, "page.html"))
		require.NoError(t, err)
		spec := AskSpec{
			Question: cover.Question, BigPicture: cover.BigPicture, Options: cover.Options,
			Recommended: cover.Recommend, RecommendationReason: cover.Reason, TechnicalDetails: cover.TechnicalDetails,
		}
		explainer := PacketExplainer{Field: "attachments[0]", Name: "page.html", MediaType: "text/html", Size: int64(len(page)), HTML: page}
		codes := []string{}
		for _, warning := range CheckPacket(spec, nil, &explainer) {
			if pageQualityCodes[warning.Code] {
				codes = append(codes, warning.Code)
			}
		}
		sort.Strings(codes)
		out[filepath.Base(dir)] = codes
	}
	return out
}

func TestThePageCheckSeparatesPagesThatTellFromPagesThatShow(t *testing.T) {
	baseline := pageQualityFindings(t, "baseline")
	require.Equal(t, map[string][]string{
		"backup-secrets":   {WarnPageNoPicture},
		"bas-purge":        {WarnPageNoPicture},
		"codex-lower-tier": {WarnPageNoPicture},
		"image-tools-nav":  {WarnPagePictureUncropped},
		"policy-batch":     {WarnPageNoPicture},
		"rebaseline":       {WarnCommandHidden, WarnPageNoPicture},
	}, baseline, "every page that tells the decision again draws a finding")
	for name, codes := range pageQualityFindings(t, "after-2") {
		require.Empty(t, codes, "%s shows the situation and draws no page finding", name)
	}
}
