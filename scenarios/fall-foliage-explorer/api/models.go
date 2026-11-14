package main

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Region struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	State           string  `json:"state"`
	Country         string  `json:"country"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	ElevationMeters *int    `json:"elevation_meters,omitempty"`
	TypicalPeakWeek *int    `json:"typical_peak_week,omitempty"`
	DataSource      string  `json:"data_source,omitempty"`
}

type ResponseMeta struct {
	Source            string `json:"source"`
	SourceDescription string `json:"source_description,omitempty"`
	RetrievedAt       string `json:"retrieved_at"`
	RowCount          int    `json:"row_count"`
	UsingFallback     bool   `json:"using_fallback"`
}

type RegionsPayload struct {
	Regions []Region     `json:"regions"`
	Meta    ResponseMeta `json:"meta"`
}

type FoliageData struct {
	RegionID          int     `json:"region_id"`
	ObservationDate   string  `json:"observation_date"`
	FoliagePercent    int     `json:"foliage_percentage"`
	ColorIntensity    int     `json:"color_intensity"`
	PeakStatus        string  `json:"peak_status"`
	PredictedPeak     string  `json:"predicted_peak,omitempty"`
	ConfidenceScore   float64 `json:"confidence_score,omitempty"`
	DataSource        string  `json:"data_source,omitempty"`
	SourceDescription string  `json:"source_description,omitempty"`
}

type UserReport struct {
	ID            int    `json:"id"`
	RegionID      int    `json:"region_id"`
	ReportDate    string `json:"report_date"`
	FoliageStatus string `json:"foliage_status"`
	Description   string `json:"description"`
	PhotoURL      string `json:"photo_url,omitempty"`
}

type TripPlan struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Regions   []int  `json:"regions"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
