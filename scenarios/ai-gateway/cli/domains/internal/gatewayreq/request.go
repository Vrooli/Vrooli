package gatewayreq

import (
	"fmt"
	"strconv"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

func FromContext(ctx cliapp.RunContext) (*sharedv1.GatewayRequest, error) {
	kind, err := parseKind(ctx.Flag("kind"))
	if err != nil {
		return nil, err
	}
	profile, err := parseProfile(ctx.Flag("profile"))
	if err != nil {
		return nil, err
	}
	privacy, err := parsePrivacy(ctx.Flag("privacy"))
	if err != nil {
		return nil, err
	}
	timeout, err := parseInt32(ctx.Flag("timeout-ms"), "timeout-ms")
	if err != nil {
		return nil, err
	}
	maxTokens, err := parseInt32(ctx.Flag("max-output-tokens"), "max-output-tokens")
	if err != nil {
		return nil, err
	}
	maxCost, err := parseFloat64(ctx.Flag("max-cost-usd"), "max-cost-usd")
	if err != nil {
		return nil, err
	}
	return &sharedv1.GatewayRequest{
		Kind:            kind,
		Role:            ctx.Flag("role"),
		Profile:         profile,
		PrivacyClass:    privacy,
		Operation:       ctx.Flag("operation"),
		Scenario:        ctx.Flag("scenario"),
		TimeoutMs:       timeout,
		MaxCostUsd:      maxCost,
		MaxOutputTokens: maxTokens,
		RequestId:       ctx.Flag("request-id"),
	}, nil
}

func parseKind(value string) (sharedv1.RequestKind, error) {
	switch normalize(value) {
	case "", "text", "text_generation", "generation":
		return sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, nil
	case "embedding", "text_embedding":
		return sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, nil
	case "extract", "structured", "structured_extraction":
		return sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION, nil
	case "image", "image_generation":
		return sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION, nil
	case "video", "video_generation":
		return sharedv1.RequestKind_REQUEST_KIND_VIDEO_GENERATION, nil
	default:
		return 0, fmt.Errorf("unknown kind %q (use text, embedding, extract, image, or video)", value)
	}
}

func parseProfile(value string) (sharedv1.Profile, error) {
	switch normalize(value) {
	case "", "local_first":
		return sharedv1.Profile_PROFILE_LOCAL_FIRST, nil
	case "local_only":
		return sharedv1.Profile_PROFILE_LOCAL_ONLY, nil
	case "remote_only":
		return sharedv1.Profile_PROFILE_REMOTE_ONLY, nil
	case "quality_first":
		return sharedv1.Profile_PROFILE_QUALITY_FIRST, nil
	case "cheap_first":
		return sharedv1.Profile_PROFILE_CHEAP_FIRST, nil
	case "privacy_sensitive":
		return sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE, nil
	default:
		return 0, fmt.Errorf("unknown profile %q (use local-only, local-first, remote-only, quality-first, cheap-first, or privacy-sensitive)", value)
	}
}

func parsePrivacy(value string) (sharedv1.PrivacyClass, error) {
	switch normalize(value) {
	case "", "internal":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, nil
	case "public":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC, nil
	case "confidential":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, nil
	case "secret":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET, nil
	default:
		return 0, fmt.Errorf("unknown privacy class %q (use public, internal, confidential, or secret)", value)
	}
}

func parseInt32(value, name string) (int32, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %w", name, err)
	}
	return int32(n), nil
}

func parseFloat64(value, name string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s must be a number: %w", name, err)
	}
	return n, nil
}

func normalize(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}
