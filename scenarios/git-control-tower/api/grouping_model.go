package main

// GroupingRulesConfig represents the stored grouping configuration for a repository.
type GroupingRulesConfig struct {
	Enabled bool           `json:"enabled"`
	Rules   []GroupingRule `json:"rules"`
}

// GroupingRule defines a single file grouping rule.
type GroupingRule struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Prefixes []string `json:"prefixes"`
	Mode     string   `json:"mode"` // "prefix" | "segment"
}
