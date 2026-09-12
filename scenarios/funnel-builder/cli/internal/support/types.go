package support

import (
	"encoding/json"
	"time"
)

type Funnel struct {
	ID          string          `json:"id"`
	TenantID    *string         `json:"tenant_id,omitempty"`
	ProjectID   *string         `json:"project_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description,omitempty"`
	Steps       []FunnelStep    `json:"steps"`
	Settings    json.RawMessage `json:"settings"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type FunnelStep struct {
	ID             string          `json:"id"`
	FunnelID       string          `json:"funnel_id"`
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	Title          string          `json:"title"`
	Content        json.RawMessage `json:"content"`
	BranchingRules json.RawMessage `json:"branching_rules,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Project struct {
	ID          string    `json:"id"`
	TenantID    *string   `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Funnels     []Funnel  `json:"funnels"`
}

type Template struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	TemplateData json.RawMessage `json:"templateData"`
	Metrics      json.RawMessage `json:"metrics"`
}

type Lead struct {
	ID        string          `json:"id"`
	Email     string          `json:"email,omitempty"`
	Phone     string          `json:"phone,omitempty"`
	Name      string          `json:"name,omitempty"`
	Data      json.RawMessage `json:"data"`
	Source    string          `json:"source,omitempty"`
	Completed bool            `json:"completed"`
	CreatedAt time.Time       `json:"created_at"`
}

type Analytics struct {
	FunnelID       string             `json:"funnelId"`
	TotalViews     int                `json:"totalViews"`
	TotalLeads     int                `json:"totalLeads"`
	CompletedLeads int                `json:"completedLeads"`
	CapturedLeads  int                `json:"capturedLeads"`
	ConversionRate float64            `json:"conversionRate"`
	CaptureRate    float64            `json:"captureRate"`
	AverageTime    float64            `json:"averageTime"`
	DropOffPoints  []AnalyticsDropOff `json:"dropOffPoints"`
	DailyStats     []AnalyticsDaily   `json:"dailyStats"`
	TrafficSources []AnalyticsTraffic `json:"trafficSources"`
}

type AnalyticsDropOff struct {
	StepID      string  `json:"stepId"`
	StepTitle   string  `json:"stepTitle"`
	Position    int     `json:"position"`
	DropOffRate float64 `json:"dropOffRate"`
	Responses   int     `json:"responses"`
	Visitors    int     `json:"visitors"`
	AvgDuration float64 `json:"avgDuration"`
}

type AnalyticsDaily struct {
	Date        string `json:"date"`
	Views       int    `json:"views"`
	Leads       int    `json:"leads"`
	Conversions int    `json:"conversions"`
}

type AnalyticsTraffic struct {
	Source     string  `json:"source"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CreateFunnelResponse struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	PreviewURL string `json:"preview_url"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
