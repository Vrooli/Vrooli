package delivery

import (
	"fmt"
	"net/url"
	"strings"
)

// ApplySettingsUpdate applies a partial operator update to storage settings.
func ApplySettingsUpdate(settings StorageSettings, update StorageSettingsUpdate) StorageSettings {
	if update.Provider != nil {
		settings.Provider = strings.TrimSpace(*update.Provider)
	}
	if update.Bucket != nil {
		settings.Bucket = strings.TrimSpace(*update.Bucket)
	}
	if update.Region != nil {
		settings.Region = strings.TrimSpace(*update.Region)
	}
	if update.Endpoint != nil {
		settings.Endpoint = strings.TrimSpace(*update.Endpoint)
	}
	if update.ForcePathStyle != nil {
		settings.ForcePathStyle = *update.ForcePathStyle
	}
	if update.DefaultPrefix != nil {
		settings.DefaultPrefix = strings.TrimSpace(*update.DefaultPrefix)
	}
	if update.SignedURLTTLSeconds != nil {
		settings.SignedURLTTLSeconds = *update.SignedURLTTLSeconds
	}
	if update.PublicBaseURL != nil {
		settings.PublicBaseURL = strings.TrimSpace(*update.PublicBaseURL)
	}
	if update.AccessKeyID != nil {
		settings.AccessKeyID = strings.TrimSpace(*update.AccessKeyID)
	}
	if update.SecretAccessKey != nil {
		settings.SecretAccessKey = strings.TrimSpace(*update.SecretAccessKey)
	}
	if update.SessionToken != nil {
		settings.SessionToken = strings.TrimSpace(*update.SessionToken)
	}
	return settings
}

func ValidateSettings(settings StorageSettings) error {
	for label, raw := range map[string]string{"endpoint": settings.Endpoint, "public_base_url": settings.PublicBaseURL} {
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("%s must be a valid URL including scheme (e.g. https://...)", label)
		}
	}
	if settings.SignedURLTTLSeconds <= 0 {
		return fmt.Errorf("signed_url_ttl_seconds must be > 0")
	}
	if settings.SignedURLTTLSeconds > 86400 {
		return fmt.Errorf("signed_url_ttl_seconds must be <= 86400")
	}
	if (strings.TrimSpace(settings.AccessKeyID) != "") != (strings.TrimSpace(settings.SecretAccessKey) != "") {
		return fmt.Errorf("access_key_id and secret_access_key must both be provided (or both left blank)")
	}
	return nil
}
