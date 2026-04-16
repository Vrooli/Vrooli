package support

// BuildResponse mirrors the response shape returned by
// POST /api/v1/ios/build. The API currently returns the structure directly
// (no {success, data} envelope), but Decode handles both cases uniformly.
type BuildResponse struct {
	Success  bool              `json:"success"`
	BuildID  string            `json:"build_id,omitempty"`
	IPAPath  string            `json:"ipa_path,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    string            `json:"error,omitempty"`
}
