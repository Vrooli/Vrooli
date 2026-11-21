package gatewayreq

import (
	"encoding/hex"
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
	privacy, err := parsePrivacy(ctx.Flag("privacy-class"))
	if err != nil {
		return nil, err
	}
	privacyProvenance, err := InferencePrivacyProvenance(ctx)
	if err != nil {
		return nil, err
	}
	limits, err := requestLimits(ctx)
	if err != nil {
		return nil, err
	}
	metadata, requiredVRAMField, err := requestCapacityMetadata(ctx, limits.requiredVRAM)
	if err != nil {
		return nil, err
	}
	// GatewayRequest.sampling stays API-only because CLI callers wrap it in an
	// envelope and the manifest resolver descends only one envelope level.
	return &sharedv1.GatewayRequest{
		Kind: kind, Role: ctx.Flag("role"), Profile: profile, PrivacyClass: privacy,
		PrivacyProvenance: privacyProvenance, Operation: ctx.Flag("operation"), Scenario: ctx.Flag("scenario"),
		TimeoutMs: limits.timeout, MaxCostUsd: limits.maxCost, MaxOutputTokens: limits.maxTokens,
		RequestId: ctx.Flag("request-id"), Metadata: metadata, RequiredVramBytes: requiredVRAMField,
	}, nil
}

type parsedLimits struct {
	timeout      int32
	maxTokens    int32
	maxCost      float64
	requiredVRAM int64
}

func requestLimits(ctx cliapp.RunContext) (parsedLimits, error) {
	var limits parsedLimits
	var err error
	if limits.timeout, err = parseInt32(ctx.Flag("timeout-ms"), "timeout-ms"); err != nil {
		return parsedLimits{}, err
	}
	if limits.maxTokens, err = parseInt32(optionalFlag(ctx, "max-output-tokens"), "max-output-tokens"); err != nil {
		return parsedLimits{}, err
	}
	if limits.maxCost, err = parseFloat64(optionalFlag(ctx, "max-cost-usd"), "max-cost-usd"); err != nil {
		return parsedLimits{}, err
	}
	limits.requiredVRAM, err = parseInt64(optionalFlag(ctx, "required-vram-bytes"), "required-vram-bytes")
	return limits, err
}

func requestCapacityMetadata(ctx cliapp.RunContext, requiredVRAM int64) (map[string]string, *int64, error) {
	expectedDigest := ""
	if ctx.FlagDeclared("expected-worker-digest") {
		expectedDigest = strings.TrimSpace(ctx.Flag("expected-worker-digest"))
	}
	if expectedDigest != "" {
		if requiredVRAM <= 0 {
			return nil, nil, fmt.Errorf("--expected-worker-digest requires a positive --required-vram-bytes")
		}
		digest := strings.TrimPrefix(strings.ToLower(expectedDigest), "sha256:")
		if len(digest) != 64 {
			return nil, nil, fmt.Errorf("--expected-worker-digest must be a SHA-256 digest")
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, nil, fmt.Errorf("--expected-worker-digest must be a SHA-256 digest: %w", err)
		}
	}
	metadata := map[string]string{}
	var requiredVRAMField *int64
	if requiredVRAM > 0 {
		requiredVRAMField = &requiredVRAM
		// Keep the metadata value for older API servers during a rolling update.
		metadata["required_vram_bytes"] = strconv.FormatInt(requiredVRAM, 10)
		metadata["capacity_allow_reclaim"] = "false"
	}
	if expectedDigest != "" {
		metadata["expected_worker_digest"] = expectedDigest
	}
	return metadata, requiredVRAMField, nil
}

// optionalFlag reads a flag only when the command's schema declares it. The
// generic GatewayRequest builder is shared by commands whose flag sets differ
// (media submission has no output-token or VRAM controls), so an undeclared
// optional flag means "not supplied", not a panic.
func optionalFlag(ctx cliapp.RunContext, name string) string {
	if !ctx.FlagDeclared(name) {
		return ""
	}
	return ctx.Flag(name)
}

func InferencePrivacyProvenance(ctx cliapp.RunContext) (string, error) {
	value := strings.TrimSpace(ctx.Flag("privacy-provenance"))
	if value == "" {
		return "", fmt.Errorf("--privacy-provenance is required and must name the non-sensitive source and basis for the privacy classification")
	}
	return value, nil
}

// InferenceContract resolves the profile and optional privacy class carried by
// typed inference commands. Omitted privacy stays UNSPECIFIED so the shared
// routing policy can keep remote candidates ineligible without inventing a
// classification. Generic GatewayRequest commands use FromContext, whose
// manifest supplies an explicit privacy default.
func InferenceContract(ctx cliapp.RunContext) (sharedv1.Profile, sharedv1.PrivacyClass, error) {
	profile, err := parseProfile(ctx.Flag("profile"))
	if err != nil {
		return sharedv1.Profile_PROFILE_UNSPECIFIED, sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED, err
	}
	privacy := sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED
	if value := strings.TrimSpace(ctx.Flag("privacy-class")); value != "" {
		privacy, err = parsePrivacy(value)
		if err != nil {
			return sharedv1.Profile_PROFILE_UNSPECIFIED, sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED, err
		}
	}
	return profile, privacy, nil
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

func parseInt64(value, name string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return n, nil
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
