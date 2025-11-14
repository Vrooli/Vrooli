package main

import (
	"regexp"
	"time"
)

const (
	BacklogStatusPending   = "pending"
	BacklogStatusConverted = "converted"
	BacklogStatusArchived  = "archived"
)

var bulletStripper = regexp.MustCompile(`^[\s\-\*•‣◦⁃∙·]+`)

// BacklogEntry represents a jot-note style backlog item before draft creation.
type BacklogEntry struct {
	ID               string    `json:"id"`
	IdeaText         string    `json:"idea_text"`
	EntityType       string    `json:"entity_type"`
	SuggestedName    string    `json:"suggested_name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ConvertedDraftID *string   `json:"converted_draft_id,omitempty"`
}

type BacklogListResponse struct {
	Entries []BacklogEntry `json:"entries"`
	Total   int            `json:"total"`
}

type BacklogCreateEntry struct {
	IdeaText      string `json:"idea_text"`
	EntityType    string `json:"entity_type"`
	SuggestedName string `json:"suggested_name"`
}

type BacklogCreateRequest struct {
	RawInput   string               `json:"raw_input"`
	EntityType string               `json:"entity_type"`
	Entries    []BacklogCreateEntry `json:"entries"`
}

type BacklogCreateResponse struct {
	Entries []BacklogEntry `json:"entries"`
}

type BacklogConvertRequest struct {
	EntryIDs []string `json:"entry_ids"`
}

type BacklogConvertResult struct {
	Entry BacklogEntry `json:"entry"`
	Draft *Draft       `json:"draft,omitempty"`
	Error string       `json:"error,omitempty"`
}

type BacklogConvertResponse struct {
	Results []BacklogConvertResult `json:"results"`
}
