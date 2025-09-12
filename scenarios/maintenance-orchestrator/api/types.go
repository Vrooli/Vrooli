package main

import "time"

type MaintenanceScenario struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName"`
	Description   string            `json:"description"`
	IsActive      bool              `json:"isActive"`
	Endpoint      string            `json:"endpoint"`
	Port          int               `json:"port"`
	Tags          []string          `json:"tags"`
	LastActive    *time.Time        `json:"lastActive,omitempty"`
	ResourceUsage map[string]float64 `json:"resourceUsage,omitempty"`
}

type Preset struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	States      map[string]bool `json:"states"`
	Tags        []string        `json:"tags,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	IsDefault   bool            `json:"isDefault"`
	IsActive    bool            `json:"isActive"`
}

type ActivityEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Scenario  string    `json:"scenario,omitempty"`
	Preset    string    `json:"preset,omitempty"`
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
}