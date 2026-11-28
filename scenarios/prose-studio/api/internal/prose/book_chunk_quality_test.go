package prose

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/textmetrics"
)

type failedChunkFixture struct {
	SourceDocumentID   string    `json:"source_document_id"`
	Brief              BookBrief `json:"brief"`
	Section            Section   `json:"section"`
	Document           Document  `json:"document"`
	CanonicalCandidate Candidate `json:"canonical_candidate"`
	Parts              []struct {
		ID           string `json:"id"`
		PromptSHA256 string `json:"prompt_sha256"`
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		Locality     string `json:"locality"`
		InputTokens  int    `json:"input_tokens"`
		OutputTokens int    `json:"output_tokens"`
		Text         string `json:"text"`
	} `json:"parts"`
}

func loadFailedChunkFixture(t *testing.T) (failedChunkFixture, []Candidate) {
	t.Helper()
	raw, err := os.ReadFile("testdata/chunk-quality-regression.json")
	require.NoError(t, err)
	var fixture failedChunkFixture
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Equal(t, "7625de94-f3dc-447f-8bd4-a4285380d8d8", fixture.SourceDocumentID)
	parts := make([]Candidate, len(fixture.Parts))
	for i, part := range fixture.Parts {
		parts[i] = Candidate{
			ID: part.ID, Text: part.Text,
			Eligibility:  Eligibility{Eligible: true},
			Measurements: textmetrics.Analyze(part.Text, nil),
			Provenance:   Provenance{Provider: part.Provider, ResolvedModelRef: part.Model, Locality: part.Locality, InputTokens: part.InputTokens, OutputTokens: part.OutputTokens, PromptSHA256: part.PromptSHA256},
		}
	}
	return fixture, parts
}

func TestFailedChunkFixtureIsRejectedForUnsupportedSpecificsAndRepeatedSetup(t *testing.T) {
	fixture, parts := loadFailedChunkFixture(t)
	require.Len(t, parts, 6)
	require.NoError(t, fixture.Brief.Validate(), "legacy v1 requests with omitted evidence policy default conservatively to unknown details")

	// The persisted prompts asked for a short interval and position fields, but
	// the typed brief supplied no concrete time, location, or source facts.
	require.Contains(t, bookBriefViolation(&fixture.Brief, parts[1].Text), "unsupported concrete time detail")
	require.Contains(t, bookBriefViolation(&fixture.Brief, parts[0].Text), "unsupported named location detail")
	require.Contains(t, unsupportedConcreteClaim("A siren passes.", nil), "unsupported concrete source")
	require.True(t, repeatsSetup(parts[1].Text, parts[:1]), "part 2 restarts the same scene setup as part 1")
	require.True(t, repeatsSetup(parts[2].Text, parts[:2]), "part 3 restarts the same scene setup as earlier parts")

	plans := bookSectionPartPlans(fixture.Section, 550, 440, nil)
	_, _, err := validateBookChunkCandidate(&fixture.Brief, plans, parts)
	require.ErrorContains(t, err, "unsupported named location detail")
}

func TestBookEvidenceBoundaryDefaultsToUnknownAndAllowsOnlyApprovedFacts(t *testing.T) {
	brief := syntheticBookBriefForChunks()
	prompt := brief.PromptText()
	for _, required := range []string{"creative direction and intent are instructions, not evidence", "Approved facts: none", "Unknown details: time, location, source_identity", "do not convert requested fields"} {
		require.Contains(t, prompt, required)
	}
	claim := "Wait sixty seconds at Fourth and Main while a siren passes."
	require.Contains(t, unsupportedConcreteClaim(claim, nil), "unsupported concrete time detail")
	require.Contains(t, unsupportedConcreteClaim("Wait 60-seconds before returning.", nil), "unsupported concrete time detail")
	require.Contains(t, unsupportedConcreteClaim("Return at dawn.", nil), "unsupported concrete time detail")
	require.Contains(t, unsupportedConcreteClaim(claim, []string{"The observation was at Fourth and Main."}), "unsupported concrete time detail")
	require.Empty(t, unsupportedConcreteClaim("Wait sixty seconds at Fourth and Main while a siren passes.", []string{"Wait sixty seconds at Fourth and Main while a siren passes."}))
	require.Contains(t, unsupportedConcreteClaim("A siren was heard.", []string{"No siren was heard."}), "unsupported concrete source")

	brief.EvidenceBoundary = &BookEvidenceBoundary{ApprovedFacts: []string{"A siren passes."}, UnknownDetails: []string{UnknownDetailTime, UnknownDetailLocation}}
	require.NoError(t, brief.Validate())
	require.Contains(t, brief.PromptText(), "Approved facts: A siren passes.")
	require.Contains(t, brief.PromptText(), "Unknown details: time, location")
	brief.EvidenceBoundary.UnknownDetails = []string{"invented-category"}
	require.ErrorIs(t, brief.Validate(), ErrInvalidBookBrief)
}

func TestBookChunkPlanPurposesAdvanceWithoutRestartingScene(t *testing.T) {
	brief := syntheticBookBriefForChunks()
	section := Section{Kind: "chapter", Intent: brief.ChapterIntent[0]}
	plans := bookSectionPartPlans(section, 550, 440, nil)
	require.Len(t, plans, 6)
	require.Contains(t, plans[0].Purpose, "Do not stage the reader")
	require.Contains(t, plans[1].Purpose, "do not restart the scene")
	require.Contains(t, plans[2].Purpose, "different practice step")
	require.Contains(t, plans[3].Purpose, "field-card")
	require.Contains(t, plans[4].Purpose, "boundary between direct observation and inference")
	require.Contains(t, plans[5].Purpose, "Do not summarize or reintroduce")
	query := bookSectionPartPrompt(brief, section, plans[2], plans, "The prior part's last observation.", 440, 660)
	require.Contains(t, query, "continue after them without repeating their setup")
	require.Contains(t, query, "The prior part's last observation.")
	require.Contains(t, query, "never restart the scene")
}

func TestRepeatedSetupIsRejectedAtCanonicalAssemblyBoundary(t *testing.T) {
	brief := syntheticBookBriefForChunks()
	section := Section{ID: "setup-repeat", Kind: "chapter", Intent: brief.ChapterIntent[0]}
	plans := bookSectionPartPlans(section, 550, 440, nil)
	parts := syntheticChunkCandidates(plans)
	for i := 0; i < 2; i++ {
		parts[i].Text = "Stand still. Close your eyes. " + fakeBookSectionPart("between 47 and 137 words", i+100)
		parts[i].Measurements = textmetrics.Analyze(parts[i].Text, nil)
		parts[i].Provenance.OutputTokens = len(strings.Fields(parts[i].Text))
	}
	_, _, err := validateBookChunkCandidate(brief, plans, parts)
	require.ErrorContains(t, err, "part 2 repeats prior scene setup or staging")
}
