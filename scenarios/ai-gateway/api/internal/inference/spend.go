package inference

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func spendAuthorizationMatches(request ProviderRequest, authorization *inferencev1.SpendAuthorization) bool {
	limit := request.SpendLimit
	if !spendAuthorizationCoreMatches(request, limit, authorization) {
		return false
	}
	return spendReferencesMatch(limit.GetReferenceSha256(), authorization.GetReferenceSha256())
}

func spendAuthorizationCoreMatches(request ProviderRequest, limit *inferencev1.SpendLimit, authorization *inferencev1.SpendAuthorization) bool {
	if limit == nil || authorization == nil {
		return false
	}
	return spendIdentityMatches(limit, authorization) && spendBoundsMatch(request, authorization) && spendMetadataPresent(authorization)
}

func spendIdentityMatches(limit *inferencev1.SpendLimit, authorization *inferencev1.SpendAuthorization) bool {
	return authorization.GetStatus() == "authorized" && authorization.GetAuthorizationId() != "" &&
		authorization.GetMaxCostMicros() == limit.GetMaxCostMicros() && authorization.GetOperationId() == limit.GetOperationId() &&
		authorization.GetInputFingerprint() == limit.GetInputFingerprint() && authorization.GetFundingRoute() == limit.GetFundingRoute() &&
		authorization.GetImageCount() == limit.GetImageCount()
}

func spendBoundsMatch(request ProviderRequest, authorization *inferencev1.SpendAuthorization) bool {
	return authorization.GetOutputTokensBound() == request.MaxOutputTokens && authorization.GetInputTokensBound() > 0 &&
		authorization.GetContextWindowTokens() >= authorization.GetInputTokensBound()+authorization.GetOutputTokensBound()
}

func spendMetadataPresent(authorization *inferencev1.SpendAuthorization) bool {
	return authorization.GetModel() != "" && authorization.GetProviderControlsSha256() != "" &&
		(authorization.GetContextSource() == "openrouter-current-endpoint-metadata" || authorization.GetContextSource() == "lpbs-current-endpoint-metadata")
}

func spendReferencesMatch(expected, actual []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, digest := range expected {
		if actual[index] != digest {
			return false
		}
	}
	return true
}

func validateGeometrySpendLimit(request ProviderRequest) error {
	geometryRole := strings.HasPrefix(strings.TrimSpace(request.Role), "device.geometry.")
	remote := request.Profile == sharedv1.Profile_PROFILE_REMOTE_ONLY
	if !geometryRole {
		if request.SpendLimit != nil {
			return fmt.Errorf("%w: spend limits are supported only for device geometry roles", ErrUnavailable)
		}
		return nil
	}
	if !remote {
		if request.SpendLimit != nil {
			return fmt.Errorf("%w: local geometry requests must not carry a remote spend limit", ErrUnavailable)
		}
		return nil
	}
	return validateRemoteGeometrySpendLimit(request)
}

func validateRemoteGeometrySpendLimit(request ProviderRequest) error {
	limit := request.SpendLimit
	if !remoteGeometryLimitMatches(request, limit) {
		return fmt.Errorf("%w: remote geometry requires a positive spend ceiling bound to operation, fingerprint, and all reference images", ErrUnavailable)
	}
	if request.MaxProviderCalls != 1 {
		return fmt.Errorf("%w: spend-limited geometry allows one provider execution", ErrUnavailable)
	}
	expectedRoute := "byok"
	if strings.TrimSpace(request.Role) == "device.geometry.vrooli_credits" {
		expectedRoute = "vrooli_credits"
	} else if strings.TrimSpace(request.Role) != "device.geometry.byok" {
		return fmt.Errorf("%w: unsupported remote geometry role", ErrUnavailable)
	}
	if limit.GetFundingRoute() != expectedRoute || !validOperationID(limit.GetOperationId()) {
		return fmt.Errorf("%w: spend authorization route or operation identity does not match the request", ErrUnavailable)
	}
	return validatePinnedGeometryReferences(request, limit)
}

func remoteGeometryLimitMatches(request ProviderRequest, limit *inferencev1.SpendLimit) bool {
	return limit != nil && limit.GetMaxCostMicros() > 0 && limit.GetOperationId() != "" && validHexDigest(limit.GetInputFingerprint()) &&
		limit.GetImageCount() >= 1 && limit.GetImageCount() <= 4 && int(limit.GetImageCount()) == len(request.Attachments) &&
		len(limit.GetReferenceSha256()) == len(request.Attachments)
}

func validatePinnedGeometryReferences(request ProviderRequest, limit *inferencev1.SpendLimit) error {
	for index, attachment := range request.Attachments {
		if attachment == nil || len(attachment.GetInlineBytes()) == 0 || attachment.GetModality() != sharedv1.Modality_MODALITY_IMAGE {
			return fmt.Errorf("%w: spend-limited geometry requires pinned inline image attachments", ErrUnavailable)
		}
		digest := sha256.Sum256(attachment.GetInlineBytes())
		if hex.EncodeToString(digest[:]) != strings.ToLower(strings.TrimSpace(limit.GetReferenceSha256()[index])) {
			return fmt.Errorf("%w: reference image digest does not match the spend limit", ErrUnavailable)
		}
	}
	return nil
}

func validHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validOperationID(value string) bool {
	return len(value) == 36 && value[8] == '-' && value[13] == '-' && value[18] == '-' && value[23] == '-' &&
		validLowerHex(value[:8]) && validLowerHex(value[9:13]) && validLowerHex(value[14:18]) &&
		validLowerHex(value[19:23]) && validLowerHex(value[24:])
}

func validLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
