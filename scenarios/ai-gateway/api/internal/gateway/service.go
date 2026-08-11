package gateway

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

var forbiddenMetadataKeys = map[string]string{
	"api_key":              "Provider credentials are resource-owned and must not enter AI Gateway requests.",
	"base_url":             "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"context_window":       "Concrete context windows are provider catalog details, not caller authority.",
	"embedding_dimensions": "Embedding dimensions are resolved through provider role policy.",
	"model":                "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"model_slug":           "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"openrouter_api_key":   "Provider credentials are resource-owned and must not enter AI Gateway requests.",
	"ollama_url":           "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"provider":             "Provider selection belongs to routing policy, not callers.",
	"provider_base_url":    "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"provider_model":       "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"provider_url":         "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"remote_model":         "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
}

// Service owns provider-neutral request validation. It deliberately has no
// provider adapter dependency; provider execution starts in later phases after
// this boundary accepts the request shape.
type Service struct{}

const (
	MaxAttachments    = 4
	MaxAttachmentSize = 10 * 1024 * 1024
)

func New() *Service { return &Service{} }

func (s *Service) Validate(req *sharedv1.GatewayRequest) []*sharedv1.ValidationIssue {
	if req == nil {
		return []*sharedv1.ValidationIssue{issue("request", "required", "request is required")}
	}
	var issues []*sharedv1.ValidationIssue
	if req.GetKind() == sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED {
		issues = append(issues, issue("kind", "required", "request kind is required"))
	}
	if strings.TrimSpace(req.GetRole()) == "" {
		issues = append(issues, issue("role", "required", "role is required"))
	}
	if req.GetProfile() == sharedv1.Profile_PROFILE_UNSPECIFIED {
		issues = append(issues, issue("profile", "required", "routing profile is required"))
	}
	if req.GetPrivacyClass() == sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		issues = append(issues, issue("privacy_class", "required", "privacy class is required"))
	}
	if req.GetTimeoutMs() < 0 {
		issues = append(issues, issue("timeout_ms", "invalid", "timeout_ms cannot be negative"))
	}
	if req.GetMaxCostUsd() < 0 {
		issues = append(issues, issue("max_cost_usd", "invalid", "max_cost_usd cannot be negative"))
	}
	if req.GetMaxOutputTokens() < 0 {
		issues = append(issues, issue("max_output_tokens", "invalid", "max_output_tokens cannot be negative"))
	}
	issues = append(issues, validateProviderSpecificMetadata(req.GetMetadata())...)
	issues = append(issues, validatePrivacyProfile(req)...)
	issues = append(issues, validateAttachments(req)...)
	return issues
}

func AcceptedProfiles() []string {
	return []string{
		"local-only",
		"local-first",
		"remote-only",
		"quality-first",
		"cheap-first",
		"privacy-sensitive",
	}
}

func validateProviderSpecificMetadata(metadata map[string]string) []*sharedv1.ValidationIssue {
	if len(metadata) == 0 {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var issues []*sharedv1.ValidationIssue
	for _, rawKey := range keys {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		key = strings.ReplaceAll(key, "-", "_")
		if msg, ok := forbiddenMetadataKeys[key]; ok {
			issues = append(issues, issue("metadata."+rawKey, "provider_specific_field", msg))
		}
	}
	return issues
}

func validatePrivacyProfile(req *sharedv1.GatewayRequest) []*sharedv1.ValidationIssue {
	privacy := req.GetPrivacyClass()
	profile := req.GetProfile()
	if privacy != sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET {
		return nil
	}
	switch profile {
	case sharedv1.Profile_PROFILE_REMOTE_ONLY,
		sharedv1.Profile_PROFILE_QUALITY_FIRST,
		sharedv1.Profile_PROFILE_CHEAP_FIRST:
		return []*sharedv1.ValidationIssue{
			issue("profile", "privacy_profile_conflict", "secret requests cannot use profiles that may require remote providers"),
		}
	default:
		return nil
	}
}

func validateAttachments(req *sharedv1.GatewayRequest) []*sharedv1.ValidationIssue {
	attachments := req.GetAttachments()
	if len(attachments) == 0 {
		return nil
	}
	var issues []*sharedv1.ValidationIssue
	if req.GetPrivacyClass() == sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC {
		issues = append(issues, issue("attachments", "public_privacy_class", "attachments require a non-public privacy class"))
	}
	if len(attachments) > MaxAttachments {
		issues = append(issues, issue("attachments", "limit_exceeded", "attachments exceed the maximum of 4"))
	}
	for index, attachment := range attachments {
		field := "attachments[" + strconv.Itoa(index) + "]"
		if attachment == nil {
			issues = append(issues, issue(field, "invalid", "attachment is required"))
			continue
		}
		if attachment.GetModality() == sharedv1.Modality_MODALITY_UNSPECIFIED {
			issues = append(issues, issue(field+".modality", "required", "attachment modality is required"))
		}
		mediaType := strings.ToLower(strings.TrimSpace(attachment.GetMediaType()))
		if attachment.GetPayload() == nil {
			issues = append(issues, issue(field+".payload", "required", "attachment must contain inline_bytes or an opaque reference"))
			continue
		}
		if reference := strings.TrimSpace(attachment.GetReference()); reference != "" {
			if strings.Contains(reference, "://") || strings.HasPrefix(strings.ToLower(reference), "data:") {
				issues = append(issues, issue(field+".reference", "opaque_reference_required", "attachment references must not contain provider URLs or data URLs"))
			}
			if attachment.GetBytes() > MaxAttachmentSize {
				issues = append(issues, issue(field+".bytes", "limit_exceeded", "attachment bytes exceed the maximum of 10485760"))
			}
			continue
		}
		data := attachment.GetInlineBytes()
		if len(data) == 0 {
			issues = append(issues, issue(field+".inline_bytes", "required", "inline attachment bytes must not be empty"))
			continue
		}
		if len(data) > MaxAttachmentSize {
			issues = append(issues, issue(field+".inline_bytes", "limit_exceeded", "attachment bytes exceed the maximum of 10485760"))
		}
		if attachment.GetBytes() != 0 && attachment.GetBytes() != uint64(len(data)) {
			issues = append(issues, issue(field+".bytes", "mismatch", "declared attachment bytes do not match inline payload length"))
		}
		if attachment.GetModality() != sharedv1.Modality_MODALITY_IMAGE {
			continue
		}
		if !supportedImageMediaType(mediaType) {
			issues = append(issues, issue(field+".media_type", "unsupported", "image attachment media_type must be image/png, image/jpeg, or image/webp"))
			continue
		}
		width, height, err := imageDimensions(mediaType, data)
		if err != nil {
			issues = append(issues, issue(field, "invalid_image_header", err.Error()))
			continue
		}
		if (attachment.GetWidth() != 0 && attachment.GetWidth() != width) || (attachment.GetHeight() != 0 && attachment.GetHeight() != height) {
			issues = append(issues, issue(field+".dimensions", "mismatch", "declared attachment dimensions do not match the image header"))
		}
	}
	return issues
}

func supportedImageMediaType(mediaType string) bool {
	switch mediaType {
	case "image/png", "image/jpeg", "image/webp":
		return true
	default:
		return false
	}
}

// imageDimensions reads only container headers. It deliberately does not
// decode pixels, keeping the gateway's untrusted-input attack surface small.
func imageDimensions(mediaType string, data []byte) (uint32, uint32, error) {
	switch mediaType {
	case "image/png":
		if len(data) < 24 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) || string(data[12:16]) != "IHDR" {
			return 0, 0, fmt.Errorf("image/png header is invalid")
		}
		return binary.BigEndian.Uint32(data[16:20]), binary.BigEndian.Uint32(data[20:24]), nil
	case "image/jpeg":
		return jpegDimensions(data)
	case "image/webp":
		return webpDimensions(data)
	default:
		return 0, 0, fmt.Errorf("unsupported image media type %q", mediaType)
	}
}

func jpegDimensions(data []byte) (uint32, uint32, error) {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 0, 0, fmt.Errorf("image/jpeg header is invalid")
	}
	for offset := 2; offset+3 < len(data); {
		if data[offset] != 0xff {
			offset++
			continue
		}
		for offset < len(data) && data[offset] == 0xff {
			offset++
		}
		if offset >= len(data) {
			break
		}
		marker := data[offset]
		offset++
		if marker == 0xd8 || marker == 0xd9 {
			continue
		}
		if offset+2 > len(data) {
			break
		}
		segmentLength := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		if segmentLength < 2 || offset+segmentLength > len(data) {
			break
		}
		if isJPEGStartOfFrame(marker) {
			if segmentLength < 7 {
				break
			}
			return uint32(binary.BigEndian.Uint16(data[offset+5 : offset+7])), uint32(binary.BigEndian.Uint16(data[offset+3 : offset+5])), nil
		}
		offset += segmentLength
	}
	return 0, 0, fmt.Errorf("image/jpeg dimensions are missing")
}

func isJPEGStartOfFrame(marker byte) bool {
	return marker >= 0xc0 && marker <= 0xc3 || marker >= 0xc5 && marker <= 0xc7 || marker >= 0xc9 && marker <= 0xcb || marker >= 0xcd && marker <= 0xcf
}

func webpDimensions(data []byte) (uint32, uint32, error) {
	if len(data) < 30 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || string(data[12:16]) != "VP8X" {
		return 0, 0, fmt.Errorf("image/webp header is invalid")
	}
	width := 1 + uint32(data[24]) | uint32(data[25])<<8 | uint32(data[26])<<16
	height := 1 + uint32(data[27]) | uint32(data[28])<<8 | uint32(data[29])<<16
	return width, height, nil
}

func issue(field, code, message string) *sharedv1.ValidationIssue {
	return &sharedv1.ValidationIssue{Field: field, Code: code, Message: message}
}
