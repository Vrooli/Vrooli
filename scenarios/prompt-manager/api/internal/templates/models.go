package templates

// AgentFileTemplateMeta defines metadata stored in template.json.
type AgentFileTemplateMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	FileName    string `json:"fileName"`
	Entry       string `json:"entry,omitempty"`
}

// AgentFileTemplate is the API response shape with content included.
type AgentFileTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	FileName    string `json:"fileName"`
	Content     string `json:"content"`
}

// AgentFileTemplateListResponse wraps template results.
type AgentFileTemplateListResponse struct {
	Templates []AgentFileTemplate `json:"templates"`
	Count     int                 `json:"count"`
}
