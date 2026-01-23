// Package ogmeta provides Open Graph metadata fetching for link previews.
package ogmeta

// Response represents the OG metadata for a URL.
type Response struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"siteName"`
	Type        string `json:"type"`
	Favicon     string `json:"favicon"`
}
