// Package contracts contains transport-neutral values shared by more than one
// product domain. It contains no persistence or business behavior.
package contracts

// VariantSEOConfig is the serialized SEO override embedded in a variant
// snapshot. Experimentation persists the snapshot and content renders the
// override; neither domain owns the other's behavior.
type VariantSEOConfig struct {
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	OGTitle        string                 `json:"og_title,omitempty"`
	OGDescription  string                 `json:"og_description,omitempty"`
	OGImageURL     string                 `json:"og_image_url,omitempty"`
	TwitterCard    string                 `json:"twitter_card,omitempty"`
	CanonicalPath  string                 `json:"canonical_path,omitempty"`
	NoIndex        bool                   `json:"noindex,omitempty"`
	StructuredData map[string]interface{} `json:"structured_data,omitempty"`
}
