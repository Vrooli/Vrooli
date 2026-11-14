package androidadb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) readNetworkState(ctx context.Context) (*strategy.NetworkState, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	read := func(args ...string) ([]byte, error) {
		return a.runner.Run(ctx, "adb", a.args(args...)...)
	}
	wifiOutput, err := read("shell", "dumpsys", "wifi")
	if err != nil {
		return nil, fmt.Errorf("read Android Wi-Fi state: %w", err)
	}
	wifiEnabled := parseWiFiEnabled(wifiOutput)
	if wifiEnabled == nil {
		return nil, fmt.Errorf("Android Wi-Fi state was not recognized")
	}
	dataOutput, err := read("shell", "settings", "get", "global", "mobile_data")
	if err != nil {
		return nil, fmt.Errorf("read Android mobile-data state: %w", err)
	}
	dataEnabled := parseAndroidBooleanSetting(dataOutput)
	if dataEnabled == nil {
		return nil, fmt.Errorf("Android mobile-data state was not recognized")
	}
	airplaneOutput, err := read("shell", "settings", "get", "global", "airplane_mode_on")
	if err != nil {
		return nil, fmt.Errorf("read Android airplane-mode state: %w", err)
	}
	airplaneEnabled := parseAndroidBooleanSetting(airplaneOutput)
	if airplaneEnabled == nil {
		return nil, fmt.Errorf("Android airplane-mode state was not recognized")
	}
	return &strategy.NetworkState{WifiEnabled: wifiEnabled, MobileDataEnabled: dataEnabled, AirplaneModeEnabled: airplaneEnabled}, nil
}

func parseAndroidBooleanSetting(value []byte) *bool {
	switch strings.TrimSpace(string(value)) {
	case "1":
		result := true
		return &result
	case "0":
		result := false
		return &result
	default:
		return nil
	}
}

func parseWiFiEnabled(value []byte) *bool {
	text := strings.ToLower(string(value))
	for _, marker := range []string{"wi-fi is enabled", "wifi is enabled", "mwifienabled=true", "mwifienabled = true", "mwifienabled: true"} {
		if strings.Contains(text, marker) {
			result := true
			return &result
		}
	}
	for _, marker := range []string{"wi-fi is disabled", "wifi is disabled", "mwifienabled=false", "mwifienabled = false", "mwifienabled: false"} {
		if strings.Contains(text, marker) {
			result := false
			return &result
		}
	}
	return nil
}
