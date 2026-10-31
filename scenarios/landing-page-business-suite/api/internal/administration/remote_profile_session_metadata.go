package administration

import (
	"net/url"
	"regexp"
	"strings"
)

// RemoteProfileSessionAgentPrefix marks user agents issued by remote-profile clients.
const RemoteProfileSessionAgentPrefix = "LPBS-RemoteProfile/1 "

var remoteProfileSessionMetaValueSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._:@/-]+`)

// RemoteProfileSessionMetadata identifies a remote-profile-originated admin session.
type RemoteProfileSessionMetadata struct {
	ConnectorID string `json:"connector_id"`
	ProfileTag  string `json:"profile_tag,omitempty"`
	Origin      string `json:"origin,omitempty"`
}

// BuildRemoteProfileSessionUserAgent serializes remote-profile session provenance.
func BuildRemoteProfileSessionUserAgent(meta RemoteProfileSessionMetadata) string {
	parts := []string{"connector_id=" + SanitizeRemoteProfileSessionMetaValue(meta.ConnectorID)}
	if value := SanitizeRemoteProfileSessionMetaValue(meta.ProfileTag); value != "" {
		parts = append(parts, "profile_tag="+value)
	}
	if value := SanitizeRemoteProfileSessionMetaValue(meta.Origin); value != "" {
		parts = append(parts, "origin="+value)
	}
	return RemoteProfileSessionAgentPrefix + strings.Join(parts, ";")
}

// ParseRemoteProfileSessionUserAgent reads remote-profile session provenance.
func ParseRemoteProfileSessionUserAgent(agent string) (RemoteProfileSessionMetadata, bool) {
	trimmed := strings.TrimSpace(agent)
	if !strings.HasPrefix(trimmed, RemoteProfileSessionAgentPrefix) {
		return RemoteProfileSessionMetadata{}, false
	}

	meta := RemoteProfileSessionMetadata{}
	for _, part := range strings.Split(strings.TrimPrefix(trimmed, RemoteProfileSessionAgentPrefix), ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		value, err := url.QueryUnescape(strings.TrimSpace(kv[1]))
		if err != nil {
			value = strings.TrimSpace(kv[1])
		}
		switch strings.ToLower(strings.TrimSpace(kv[0])) {
		case "connector_id":
			meta.ConnectorID = SanitizeRemoteProfileSessionMetaValue(value)
		case "profile_tag":
			meta.ProfileTag = SanitizeRemoteProfileSessionMetaValue(value)
		case "origin":
			meta.Origin = SanitizeRemoteProfileSessionMetaValue(value)
		}
	}
	if meta.ConnectorID == "" {
		return RemoteProfileSessionMetadata{}, false
	}
	return meta, true
}

// SanitizeRemoteProfileSessionMetaValue keeps encoded session provenance header-safe.
func SanitizeRemoteProfileSessionMetaValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return remoteProfileSessionMetaValueSanitizer.ReplaceAllString(trimmed, "_")
}
