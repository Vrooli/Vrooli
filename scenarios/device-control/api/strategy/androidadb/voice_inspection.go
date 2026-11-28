package androidadb

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"device-control/strategy"
)

var (
	inspectionPackagePattern = regexp.MustCompile(`^[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)+$`)
	inputTypePattern         = regexp.MustCompile(`\bAUDIO_DEVICE_IN_[A-Z0-9_]+\b`)
)

func (a *Adapter) inspectVoice(ctx context.Context) *strategy.VoiceInspection {
	v := &strategy.VoiceInspection{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) (string, bool) {
		b, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil || len(b) > 128*1024 {
			v.Unavailable[field] = "read-only probe failed or exceeded size bound"
			return "", false
		}
		s := strings.TrimSpace(string(b))
		if strings.HasPrefix(s, "Error:") || strings.HasPrefix(s, "Unknown command") || strings.HasPrefix(s, "Can't find service") {
			v.Unavailable[field] = "read-only probe unsupported"
			return "", false
		}
		return s, true
	}
	if value, ok := probe("microphone_feature", "shell", "pm", "list", "features"); ok {
		count, microphone := 0, false
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "feature:") {
				count++
				feature := strings.TrimPrefix(line, "feature:")
				if feature == "android.hardware.microphone" || strings.HasPrefix(feature, "android.hardware.microphone=") {
					microphone = true
				}
			}
		}
		if count > 0 {
			v.MicrophoneFeature = &microphone
		} else {
			v.Unavailable["microphone_feature"] = "feature list was not recognizable"
		}
	}

	user, userOK := probe("user", "shell", "am", "get-current-user")
	// The fixed probes below explicitly qualify the system user. Do not combine
	// role state from one user with settings from another on multi-user devices.
	systemUser := userOK && user == "0"
	if !systemUser {
		v.Unavailable["assistant_role_holders"] = "current user is not qualified"
		v.Unavailable["voice_interaction_services"] = "current user is not qualified"
		v.Unavailable["user"] = "voice configuration requires a verified system-user session"
	}
	if value, ok := probe("assistant_role_holders", "shell", "cmd", "role", "get-role-holders", "--user", "0", "android.app.role.ASSISTANT"); ok && systemUser {
		valid := true
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !inspectionPackagePattern.MatchString(line) {
				valid = false
				break
			}
			v.AssistantRoleHolders = append(v.AssistantRoleHolders, line)
		}
		if !valid {
			v.AssistantRoleHolders = nil
			v.Unavailable["assistant_role_holders"] = "role result was not recognizable"
		}
	}
	component := func(field, key string) *string {
		if !systemUser {
			v.Unavailable[field] = "current user is not qualified"
			return nil
		}
		value, ok := probe(field, "shell", "settings", "--user", "0", "get", "secure", key)
		if !ok {
			return nil
		}
		if value == "null" {
			value = ""
		} else if !launchComponentPattern.MatchString(value) {
			v.Unavailable[field] = "component result was not recognizable"
			return nil
		}
		return &value
	}
	v.AssistantComponent = component("assistant_component", "assistant")
	v.VoiceInteractionComponent = component("voice_interaction_component", "voice_interaction_service")
	if value, ok := probe("voice_interaction_services", "shell", "cmd", "package", "query-services", "--brief", "--user", "0", "-a", "android.service.voice.VoiceInteractionService"); ok && systemUser {
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if launchComponentPattern.MatchString(line) {
				v.VoiceInteractionServices = append(v.VoiceInteractionServices, line)
			}
		}
		if len(v.VoiceInteractionServices) == 0 && !strings.Contains(value, "No services found") {
			v.Unavailable["voice_interaction_services"] = "service query did not yield recognizable components"
		}
	}
	if value, ok := probe("policy_input_types", "shell", "dumpsys", "media.audio_policy"); ok {
		unique := map[string]bool{}
		for _, kind := range inputTypePattern.FindAllString(value, -1) {
			unique[kind] = true
		}
		for kind := range unique {
			v.PolicyInputTypes = append(v.PolicyInputTypes, kind)
		}
		sort.Strings(v.PolicyInputTypes)
		if len(v.PolicyInputTypes) == 0 {
			v.Unavailable["policy_input_types"] = "OEM policy did not expose symbolic input type names"
		}
	}
	return v
}
