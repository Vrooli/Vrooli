package main

import (
	"net/url"
	"regexp"
	"strings"
)

const (
	remoteProfileSessionAgentPrefix = "LPBS-RemoteProfile/1 "
)

var remoteProfileSessionMetaValueSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._:@/-]+`)

type RemoteProfileSessionMetadata struct {
	ConnectorID string `json:"connector_id"`
	ProfileTag  string `json:"profile_tag,omitempty"`
	Origin      string `json:"origin,omitempty"`
}

func buildRemoteProfileSessionUserAgent(meta RemoteProfileSessionMetadata) string {
	parts := []string{
		"connector_id=" + sanitizeRemoteProfileSessionMetaValue(meta.ConnectorID),
	}
	if value := sanitizeRemoteProfileSessionMetaValue(meta.ProfileTag); value != "" {
		parts = append(parts, "profile_tag="+value)
	}
	if value := sanitizeRemoteProfileSessionMetaValue(meta.Origin); value != "" {
		parts = append(parts, "origin="+value)
	}
	return remoteProfileSessionAgentPrefix + strings.Join(parts, ";")
}

func parseRemoteProfileSessionUserAgent(agent string) (RemoteProfileSessionMetadata, bool) {
	trimmed := strings.TrimSpace(agent)
	if !strings.HasPrefix(trimmed, remoteProfileSessionAgentPrefix) {
		return RemoteProfileSessionMetadata{}, false
	}

	payload := strings.TrimPrefix(trimmed, remoteProfileSessionAgentPrefix)
	meta := RemoteProfileSessionMetadata{}
	for _, part := range strings.Split(payload, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(kv[0]))
		value, err := url.QueryUnescape(strings.TrimSpace(kv[1]))
		if err != nil {
			value = strings.TrimSpace(kv[1])
		}
		value = sanitizeRemoteProfileSessionMetaValue(value)
		switch key {
		case "connector_id":
			meta.ConnectorID = value
		case "profile_tag":
			meta.ProfileTag = value
		case "origin":
			meta.Origin = value
		}
	}

	if meta.ConnectorID == "" {
		return RemoteProfileSessionMetadata{}, false
	}
	return meta, true
}

func sanitizeRemoteProfileSessionMetaValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return remoteProfileSessionMetaValueSanitizer.ReplaceAllString(trimmed, "_")
}
