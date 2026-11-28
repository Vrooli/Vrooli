package variant

// SEOConfigJSON is the JSON shape stored in variants.seo_config. It is the
// canonical serialization shared by the variant domain (which reads/writes the
// column during Get/Import) and the seo domain (which stores overrides and
// resolves head payloads). Field tags match the persisted on-disk form.
type SEOConfigJSON struct {
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
