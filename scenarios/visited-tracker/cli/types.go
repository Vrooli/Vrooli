package main

type campaign struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	FromAgent       string   `json:"from_agent"`
	Description     *string  `json:"description"`
	Patterns        []string `json:"patterns"`
	Location        *string  `json:"location"`
	Tag             *string  `json:"tag"`
	Notes           *string  `json:"notes"`
	TotalFiles      int      `json:"total_files"`
	VisitedFiles    int      `json:"visited_files"`
	CoveragePercent float64  `json:"coverage_percent"`
}

type campaignListResponse struct {
	Campaigns []campaign `json:"campaigns"`
	Count     int        `json:"count"`
}

type findOrCreateResponse struct {
	Created  bool     `json:"created"`
	Campaign campaign `json:"campaign"`
}

type visitResponse struct {
	Recorded  int      `json:"recorded"`
	Files     []string `json:"files"`
	Unmatched []string `json:"unmatched_patterns"`
}

type excludeResponse struct {
	ExcludedCount int      `json:"excluded_count"`
	Files         []string `json:"files"`
	Unmatched     []string `json:"unmatched_patterns"`
}

type adjustVisitResponse struct {
	FileID     string `json:"file_id"`
	VisitCount int    `json:"visit_count"`
	Action     string `json:"action"`
}

type syncResponse struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Moved   int `json:"moved"`
	Total   int `json:"total"`
}

type coverageResponse struct {
	TotalFiles       int     `json:"total_files"`
	VisitedFiles     int     `json:"visited_files"`
	UnvisitedFiles   int     `json:"unvisited_files"`
	CoveragePercent  float64 `json:"coverage_percentage"`
	AverageVisits    float64 `json:"average_visits"`
	AverageStaleness float64 `json:"average_staleness"`
}

type trackedFile struct {
	ID             string  `json:"id"`
	FilePath       string  `json:"file_path"`
	AbsolutePath   string  `json:"absolute_path"`
	VisitCount     int     `json:"visit_count"`
	StalenessScore float64 `json:"staleness_score"`
	Excluded       bool    `json:"excluded"`
	PriorityWeight float64 `json:"priority_weight"`
	Notes          *string `json:"notes"`
}

type importResponse struct {
	Message  string   `json:"message"`
	Campaign campaign `json:"campaign"`
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Readiness bool   `json:"readiness"`
	Metrics   struct {
		UptimeSeconds float64 `json:"uptime_seconds"`
	} `json:"metrics"`
}
