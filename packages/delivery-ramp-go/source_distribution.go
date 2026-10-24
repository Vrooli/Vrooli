package deliveryramp

// SourceDistributionRef is the small cross-scenario projection shared by
// source-ramp entrypoints. The source scenario owns the full record; GCT and
// Deployment Manager may link to this identity but must not store a second
// recipe or publication state.
type SourceDistributionRef struct {
	DistributionID string `json:"distribution_id"`
	Scenario       string `json:"scenario"`
	SourceDigest   string `json:"source_digest"`
	ArtifactDigest string `json:"artifact_digest"`
	Standing       string `json:"standing"`
	CanonicalURL   string `json:"canonical_url,omitempty"`
}

func (r SourceDistributionRef) Valid() bool {
	return r.DistributionID != "" && r.Scenario != "" && r.SourceDigest != "" && r.ArtifactDigest != ""
}
