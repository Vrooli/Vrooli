package main

import (
	"errors"
	"strings"
)

const (
	defaultKnowledgeCollection = "knowledge_chunks_v1"
	defaultKnowledgeVisibility = "shared"
	knowledgeSchemaVersion     = "ko.knowledge.v1"
)

func normalizeKnowledgeCollection(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultKnowledgeCollection
	}
	return value
}

func normalizeVisibility(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultKnowledgeVisibility, nil
	}
	switch value {
	case "private", "shared", "global":
		return value, nil
	default:
		return "", errors.New("visibility must be one of: private, shared, global")
	}
}

