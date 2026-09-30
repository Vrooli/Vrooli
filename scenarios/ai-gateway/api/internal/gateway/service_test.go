package gateway_test

import (
	"encoding/binary"
	"testing"

	"ai-gateway/internal/gateway"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestValidateAcceptsProviderNeutralRequest(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		TimeoutMs:    5000,
		Metadata: map[string]string{
			"trace_id": "req-1",
		},
	})
	require.Empty(t, issues)
}

func TestValidateAcceptsProviderNeutralMediaRequests(t *testing.T) { // [REQ:AIGW-MEDIA-CONTRACT]
	svc := gateway.New()
	for _, kind := range []sharedv1.RequestKind{
		sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION,
		sharedv1.RequestKind_REQUEST_KIND_VIDEO_GENERATION,
	} {
		t.Run(kind.String(), func(t *testing.T) {
			issues := svc.Validate(&sharedv1.GatewayRequest{
				Kind:         kind,
				Role:         "media.generate",
				Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
				PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
				Metadata:     map[string]string{"correlation_id": "media-1"},
			})
			require.Empty(t, issues)
		})
	}
}

func TestValidateRejectsProviderSpecificMetadata(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	svc := gateway.New()
	issues := svc.Validate(validRequest(map[string]string{
		"provider":             "openrouter",
		"model_slug":           "provider/model",
		"embedding-dimensions": "1536",
		"trace_id":             "req-1",
	}))
	require.ElementsMatch(t, []string{
		"metadata.provider",
		"metadata.model_slug",
		"metadata.embedding-dimensions",
	}, fields(issues))
	for _, got := range issues {
		require.Equal(t, "provider_specific_field", got.GetCode())
	}
}

func TestValidateRejectsSecretRemoteProfile(t *testing.T) { // [REQ:AIGW-POLICY-CONSTRAINTS]
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
	})
	require.Len(t, issues, 1)
	require.Equal(t, "profile", issues[0].GetField())
	require.Equal(t, "privacy_profile_conflict", issues[0].GetCode())
}

func TestValidateRejectsPublicImageAndAcceptsHeaderMatchedPrivateImage(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	svc := gateway.New()
	data := pngHeader(320, 200)
	attachment := &sharedv1.Attachment{
		Modality:  sharedv1.Modality_MODALITY_IMAGE,
		MediaType: "image/png",
		Width:     320,
		Height:    200,
		Bytes:     uint64(len(data)),
		Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: data},
	}
	private := validRequest(nil)
	private.Attachments = []*sharedv1.Attachment{attachment}
	require.Empty(t, svc.Validate(private))

	public := validRequest(nil)
	public.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC
	public.Attachments = []*sharedv1.Attachment{attachment}
	issues := svc.Validate(public)
	require.Contains(t, fields(issues), "attachments")
	require.Contains(t, codes(issues), "public_privacy_class")
}

func TestValidateRejectsImageDimensionMismatchAndSizeCeiling(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	svc := gateway.New()
	data := pngHeader(320, 200)
	attachment := &sharedv1.Attachment{
		Modality:  sharedv1.Modality_MODALITY_IMAGE,
		MediaType: "image/png",
		Width:     321,
		Height:    200,
		Bytes:     uint64(len(data)),
		Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: data},
	}
	issues := svc.Validate(validWithAttachment(attachment))
	require.Contains(t, codes(issues), "mismatch")

	attachment.Width = 320
	attachment.Bytes = gateway.MaxAttachmentSize + 1
	attachment.Payload = &sharedv1.Attachment_InlineBytes{InlineBytes: make([]byte, gateway.MaxAttachmentSize+1)}
	issues = svc.Validate(validWithAttachment(attachment))
	require.Contains(t, codes(issues), "limit_exceeded")
}

func TestValidateRejectsMissingRequiredFieldsAndNegativeBounds(t *testing.T) {
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		TimeoutMs:       -1,
		MaxCostUsd:      -0.01,
		MaxOutputTokens: -1,
	})
	require.ElementsMatch(t, []string{
		"kind",
		"role",
		"profile",
		"privacy_class",
		"timeout_ms",
		"max_cost_usd",
		"max_output_tokens",
	}, fields(issues))
}

func validRequest(metadata map[string]string) *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		Metadata:     metadata,
	}
}

func fields(issues []*sharedv1.ValidationIssue) []string {
	out := make([]string, 0, len(issues))
	for _, got := range issues {
		out = append(out, got.GetField())
	}
	return out
}

func codes(issues []*sharedv1.ValidationIssue) []string {
	out := make([]string, 0, len(issues))
	for _, got := range issues {
		out = append(out, got.GetCode())
	}
	return out
}

func validWithAttachment(attachment *sharedv1.Attachment) *sharedv1.GatewayRequest {
	req := validRequest(nil)
	req.Attachments = []*sharedv1.Attachment{attachment}
	return req
}

func pngHeader(width, height uint32) []byte {
	data := make([]byte, 24)
	copy(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10})
	copy(data[12:16], []byte("IHDR"))
	binary.BigEndian.PutUint32(data[16:20], width)
	binary.BigEndian.PutUint32(data[20:24], height)
	return data
}
