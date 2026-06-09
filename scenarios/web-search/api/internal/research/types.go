package research

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

// Synthesis is the always-cited L2 output. Abstained signals the fetched pages
// were insufficient or in conflict; Text is then an explicit abstention note.
type Synthesis struct {
	Text      string
	Citations []Citation
	Abstained bool
}

// L2Outcome is the full L2 result: the Brief + synthesis flags + (when capture
// was requested) the ids of the findings written.
type L2Outcome struct {
	Brief              Brief
	Abstained          bool
	CapturedFindingIDs []string
}
