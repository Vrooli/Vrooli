package topics

// CreateRequest is the request body for creating a topic.
type CreateRequest struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ParentTopicID *string  `json:"parentTopicId,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Content       string   `json:"content,omitempty"`
}

// UpdateRequest is the request body for updating a topic.
type UpdateRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	ParentTopicID *string  `json:"parentTopicId,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Icon          *string  `json:"icon,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Content       *string  `json:"content,omitempty"`
}

// Response is the JSON response for a topic.
type Response struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ParentTopicID *string  `json:"parentTopicId,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

// AccumulatedSkillsResponse is the response for accumulated skills from a topic and its ancestors.
type AccumulatedSkillsResponse struct {
	TopicID  string   `json:"topicId"`
	Ancestry []string `json:"ancestry"`
	Skills   []string `json:"skills"`
}

// MatchRequest is the request body for AI topic matching.
type MatchRequest struct {
	Queries []string `json:"queries"`
	Limit   int      `json:"limit,omitempty"`
}

// MatchedTopic represents a topic matched by AI search.
type MatchedTopic struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	ParentTopicID *string `json:"parentTopicId,omitempty"`
	Score         float64 `json:"score"`
	ScorePercent  int     `json:"scorePercent"`
}

// MatchResponse is the response for topic matching.
type MatchResponse struct {
	Topics []MatchedTopic `json:"topics"`
	Skills []string       `json:"skills"`
	Method string         `json:"method"`
}
