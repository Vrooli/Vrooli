package main

import "strings"

func normalizeEnum(value, prefix string) string {
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, prefix)
	if value == "UNSPECIFIED" {
		return ""
	}
	return strings.ToLower(value)
}

func normalizeEventType(value string) string {
	if value == "RUN_EVENT_TYPE_STATUS" {
		return "status_change"
	}
	return normalizeEnum(value, "RUN_EVENT_TYPE_")
}

// wireUploadAttachmentResponse remains because multipart upload is currently
// the only Agent Manager boundary without a protobuf RPC.
type wireUploadAttachmentResponse struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	StoragePath string `json:"storage_path"`
	URL         string `json:"url"`
}
