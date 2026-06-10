package research

import "web-search/internal/livesearch"

// Levels recorded on a Brief.
const (
	LevelL2 = "l2"
	LevelL3 = "l3"
)

// DefaultTopN is the candidate-page count used when an L2 request omits a
// positive top_n.
const DefaultTopN = 5

// MaxTopN caps how many pages an L2 request may fetch and read.
const MaxTopN = 10

// Citation grounds a synthesis claim back to the fetched Document that supports
// it.
type Citation struct {
	ResultIndex int
	URL         string
	Title       string
}

// Brief is the L2/L3 research artifact: the query, the level, the synthesized
// summary, and the citations that ground it.
type Brief struct {
	Query     string
	Level     string
	Summary   string
	Citations []Citation
}

// Document is one fetched candidate page: its source URL/title plus the
// extracted readable text the synthesizer reads.
type Document struct {
	URL   string
	Title string
	Text  string
}

// AbstainReason explains which collapse produced an L2 abstention. The single
// "sources insufficient or disagree" note used to hide four distinct causes;
// the reason makes each observable so an operator can tell a fetch problem
// from a model problem (OT-P1-001 observability).
type AbstainReason string

const (
	// ReasonNoCandidates: the L0 candidate search returned no usable pages
	// (also used when the L2 seams are unwired — nothing was ever searched).
	ReasonNoCandidates AbstainReason = "no_candidates"
	// ReasonAllFetchesEmpty: candidates existed but every fetch failed or
	// extracted no readable text.
	ReasonAllFetchesEmpty AbstainReason = "all_fetches_empty"
	// ReasonModelAbstained: the model judged the documents insufficient or in
	// conflict (or returned empty text).
	ReasonModelAbstained AbstainReason = "model_abstained"
	// ReasonReplyUnparseable: the model reply carried no parseable JSON object.
	ReasonReplyUnparseable AbstainReason = "reply_unparseable"
	// ReasonCitationsInvalid: the model answered but grounded it in no valid
	// document index — treated as fabrication by the always-cited contract.
	ReasonCitationsInvalid AbstainReason = "citations_invalid"
)

// Synthesis is the always-cited L2 output. Abstained signals the fetched pages
// were insufficient or in conflict; Text is then an explicit abstention note
// and AbstainReason carries which collapse produced it.
type Synthesis struct {
	Text          string
	Citations     []Citation
	Abstained     bool
	AbstainReason AbstainReason
}

// DocumentExcerpt records what was actually sent to the synthesis model for
// one fetched document (post excerpting/truncation, response-capped) — the
// observability needed to debug a wrong or abstaining synthesis.
type DocumentExcerpt struct {
	URL     string
	Title   string
	Excerpt string
}

// L2Outcome is the full L2 result: the Brief + synthesis flags + (when capture
// was requested) the ids of the findings written.
type L2Outcome struct {
	Brief              Brief
	Abstained          bool
	AbstainReason      AbstainReason
	CapturedFindingIDs []string
	// Excerpts mirrors, per fetched document, the text actually sent to the
	// synthesis model (capped for transport).
	Excerpts []DocumentExcerpt
	// DegradedEngines lists upstream engines that did not answer the L0
	// candidate query backing this run (partial-inputs signal).
	DegradedEngines []livesearch.EngineIssue
}
