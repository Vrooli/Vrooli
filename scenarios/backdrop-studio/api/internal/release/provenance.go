package release

// Disclosure facts, and where they come from.
//
// A model-backed backdrop cannot be released without naming the model, prompt
// and seed that produced it. The question this file answers is *who says so*.
//
// The obvious answer — put the fields on the release request and let the caller
// fill them in — is the wrong one. It makes the disclosure an assertion by
// whoever called the API, which means a caller can release a synthetic image
// labelled as anything it likes, and a caller that simply forgets a field turns
// a compliance record into a shrug. Neither failure is visible afterwards.
//
// So the facts come from the render that produced the candidate. Backdrop
// Studio recorded them at the only moment they existed — image-tools' router
// reports the model on the submit response — and the release path reads them
// back by candidate id. The caller chooses *what* to release; it does not get
// to describe how it was made.

// Provenance is what the producing render knows about one candidate.
type Provenance struct {
	Strategy    string
	ModelBacked bool
	Model       string
	Tier        string
	Prompt      string
	Negative    string
	Seed        string
	// Conditioner names the ControlNet preprocessor a guided render used, and
	// is empty for every other strategy. It is the conditioning kind Asset
	// Studio records, and the reason that field had to be generic: a scaffold
	// is not an identity.
	Conditioner string
	Parameters  string
}

// ProvenanceSource resolves a candidate's provenance by id.
//
// It is an interface rather than a direct dependency on the render store so
// this package keeps one job — the release decision — and so a unit test can
// state a candidate's provenance without building a render.
type ProvenanceSource interface {
	CandidateProvenance(candidateID string) (Provenance, bool)
}
