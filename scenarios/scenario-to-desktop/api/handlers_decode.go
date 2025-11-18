package main

import "encoding/json"

func decodeDesktopConfig(payload []byte) (*DesktopConfig, error) {
	type raw DesktopConfig
	var cfg raw
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return nil, err
	}
	result := DesktopConfig(cfg)
	var legacy struct {
		AutoManageTier1 *bool  `json:"auto_manage_tier1"`
		ServerURL       string `json:"server_url"`
		APIURL          string `json:"api_url"`
	}
	_ = json.Unmarshal(payload, &legacy)

	if !result.AutoManageVrooli && legacy.AutoManageTier1 != nil {
		result.AutoManageVrooli = *legacy.AutoManageTier1
	}
	if result.ProxyURL == "" {
		switch {
		case result.ExternalServerURL != "":
			result.ProxyURL = result.ExternalServerURL
		case legacy.ServerURL != "":
			result.ProxyURL = legacy.ServerURL
		}
	}
	if result.ProxyURL != "" {
		if normalized, err := normalizeProxyURL(result.ProxyURL); err == nil {
			result.ProxyURL = normalized
		}
		if result.ExternalServerURL == "" {
			result.ExternalServerURL = result.ProxyURL
		}
		if result.ExternalAPIURL == "" {
			result.ExternalAPIURL = proxyAPIURL(result.ProxyURL)
		}
	} else {
		if result.ExternalAPIURL == "" && legacy.APIURL != "" {
			result.ExternalAPIURL = legacy.APIURL
		}
	}
	return &result, nil
}
