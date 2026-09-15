package presentationqualification

import (
	"fmt"
	"strings"
)

const (
	testGenieValidationPrefix      = "test-genie:validation:"
	deploymentManagerReleasePrefix = "deployment-manager:release:"
)

// ParseValidationRef accepts only test-genie:validation:<opaque-id>. The ID
// is sent to Test Genie as an exact GetValidation receipt_id; paths, URLs,
// arbitrary evidence references, and untyped IDs are rejected.
func ParseValidationRef(value string) (string, error) {
	return parseTypedRef(value, testGenieValidationPrefix, "Test Genie validation")
}

// ParseReleaseRef accepts only deployment-manager:release:<opaque-id>. The ID
// is sent to Deployment Manager as an exact ReleasesService.Get release_id.
func ParseReleaseRef(value string) (string, error) {
	return parseTypedRef(value, deploymentManagerReleasePrefix, "Deployment Manager release")
}

func parseTypedRef(value, prefix, label string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, prefix) {
		return "", fmt.Errorf("%s reference must use %s<id>", label, prefix)
	}
	id := strings.TrimPrefix(value, prefix)
	if id == "" || id != strings.TrimSpace(id) || strings.ContainsAny(id, "/\\:") || strings.Contains(id, "..") || strings.ContainsAny(id, " \t\r\n") {
		return "", fmt.Errorf("%s reference has an invalid opaque id", label)
	}
	return id, nil
}
