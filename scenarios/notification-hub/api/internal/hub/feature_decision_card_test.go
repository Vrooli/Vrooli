package hub

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func validFeatureDecisionCard() featureCard {
	var card featureCard
	card.Schema = "research-feature-decision/v2"
	card.ID = "FEATURE_LOSS_BOTTOM_NAV_SETTINGS"
	card.FeatureID = "notification-hub:/settings:bottom-nav:settings"
	card.Title = "Settings in the mobile bottom navigation"
	card.Location.Screen = "/settings"
	card.Location.Taps = []string{"Open the app", "Tap Settings"}
	card.Location.Component = "BottomNavigation"
	card.Location.Source = "ui/src/navigation/BottomNavigation.tsx"
	card.Summary = "Settings disappeared from the served mobile navigation after this redesign."
	card.Evidence = append(card.Evidence, struct {
		Alt   string `json:"alt"`
		Image string `json:"image"`
	}{
		Alt:   "Before: Settings appears in the bottom navigation.",
		Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n")),
	})
	card.Options = []struct {
		Key    string `json:"key"`
		Label  string `json:"label"`
		Effect string `json:"effect"`
	}{
		{Key: "keep", Label: "Keep Settings", Effect: "Restore Settings in the mobile navigation and verify it in the served app."},
		{Key: "drop", Label: "Drop Settings", Effect: "Delete the code that provides this navigation item and verify it is gone."},
	}
	card.Recommendation = "keep"
	card.RecommendationReason = "The Hub product record says mobile users must be able to reach Settings."
	card.Terms = []struct {
		Word    string `json:"word"`
		Meaning string `json:"meaning"`
	}{{Word: "knip", Meaning: "A code scanner that reports code with no known use; its report is evidence to review, not proof to delete."}}
	card.Status = "pending"
	card.Finding.Version = "sha256:" + strings.Repeat("c", 64)
	card.Finding.BeforeBuild = "sha256:" + strings.Repeat("a", 64)
	card.Finding.AfterBuild = "sha256:" + strings.Repeat("b", 64)
	card.Finding.Kind = "navigation-item"
	card.Finding.Component = "BottomNavigation"
	card.Finding.Source = "ui/src/navigation/BottomNavigation.tsx"
	subjectHash := sha256.Sum256([]byte(card.ID + "\x00" + card.FeatureID + "\x00" + card.Finding.Version))
	card.SubjectKey = "feature-loss:" + hex.EncodeToString(subjectHash[:])
	return card
}

func renderedFeatureDecisionPage(card featureCard) *PacketExplainer {
	image := card.Evidence[0].Image
	page := `<main data-research-feature-card="v2" data-card-id="` + card.ID + `" data-feature-id="` + card.FeatureID + `" data-finding-version="` + card.Finding.Version + `" data-before-build="` + card.Finding.BeforeBuild + `" data-after-build="` + card.Finding.AfterBuild + `">` +
		`<h1>` + card.Title + `</h1><p>` + card.Summary + `</p><p>` + card.RecommendationReason + `</p><img src="` + image + `" alt="` + card.Evidence[0].Alt + `">` +
		`<section><h2>` + card.Options[0].Label + `</h2><p>` + card.Options[0].Effect + `</p><button data-od-answer="keep">Choose keep</button></section>` +
		`<section><h2>` + card.Options[1].Label + `</h2><p>` + card.Options[1].Effect + `</p><button data-od-answer="drop">Choose drop</button></section>` +
		`<h2>Words used</h2><dt>` + card.Terms[0].Word + `</dt><dd>` + card.Terms[0].Meaning + `</dd></main>`
	return &PacketExplainer{HTML: []byte(page)}
}

func validFeatureCardAsk(card featureCard, attachments ...AskAttachment) AskSpec {
	return AskSpec{
		Question:             "Keep or drop “" + card.Title + "”?",
		Options:              []AskOption{{Key: "keep", Label: card.Options[0].Label}, {Key: "drop", Label: card.Options[1].Label}},
		Recommended:          card.Recommendation,
		RecommendationReason: card.RecommendationReason,
		SubjectKey:           featureCardSubjectKey(card),
		Attachments:          attachments,
	}
}

func TestValidateFeatureDecisionCardRequiresFullFindingAndPhoneDecision(t *testing.T) {
	card := validFeatureDecisionCard()
	require.Empty(t, validateFeatureCard(card))
	require.Empty(t, validateFeatureCardPage(card, renderedFeatureDecisionPage(card)))
	require.Empty(t, validateFeatureCardAsk(card, validFeatureCardAsk(card)))
	require.NotEmpty(t, validateFeatureCardPage(card, &PacketExplainer{HTML: []byte(`<html><body>Keep or drop?</body></html>`)}))
	ask := validFeatureCardAsk(card)
	ask.SubjectKey = "another-finding"
	require.NotEmpty(t, validateFeatureCardAsk(card, ask))

	card.Finding.AfterBuild = ""
	card.RecommendationReason = ""
	card.Evidence = nil
	problems := validateFeatureCard(card)
	require.Contains(t, problems, "finding must bind full measured before_build and after_build identities")
	require.Contains(t, problems, "recommendation_reason must explain the recommended choice")
	require.Contains(t, problems, "at least one embedded before/after or component/crawl image is required")
}

func TestCheckFeatureDecisionCardsWarnsForLegacyAndRejectsIncompleteV2(t *testing.T) {
	legacy := map[string]any{"schema": "research-feature-decision/v1"}
	legacyJSON, err := json.Marshal(legacy)
	require.NoError(t, err)
	complete, err := json.Marshal(validFeatureDecisionCard())
	require.NoError(t, err)
	incomplete := validFeatureDecisionCard()
	incomplete.Finding.BeforeBuild = ""
	incompleteJSON, err := json.Marshal(incomplete)
	require.NoError(t, err)

	contents := map[string][]byte{"legacy": legacyJSON, "complete": complete, "incomplete": incompleteJSON}
	read := func(_ context.Context, sha string, max int64) ([]byte, error) { return contents[sha], nil }
	attachments := []AskAttachment{
		{Name: "legacy-card.json", Kind: AttachmentFile, SHA256: "legacy"},
		{Name: "complete-card.json", Kind: AttachmentFile, SHA256: "complete"},
		{Name: "incomplete-card.json", Kind: AttachmentFile, SHA256: "incomplete"},
		{Name: "ordinary.json", Kind: AttachmentFile, SHA256: "ordinary"},
	}
	card := validFeatureDecisionCard()
	spec := validFeatureCardAsk(card, attachments...)
	warnings := checkFeatureDecisionCards(context.Background(), spec, renderedFeatureDecisionPage(card), read)
	require.Len(t, warnings, 5)
	require.Equal(t, WarnFeatureCardInvalid, warnings[0].Code)
	require.Equal(t, WarnFeatureCardInvalid, warnings[1].Code)
	require.Equal(t, WarnFeatureCardIncomplete, warnings[2].Code)
	require.Equal(t, WarnFeatureCardInvalid, warnings[3].Code)
	require.Equal(t, "advice", warnings[0].Severity)
	require.Equal(t, WarnFeatureCardIncomplete, warnings[4].Code)
}
