// Package compat exposes workflow-definition normalization to BAS clients.
//
// Workflow JSON is intentionally authored in the concise schema form. Before
// protobuf decoding, each entry point must apply the same normalization so the
// CLI, capture service, and API accept identical workflow files.
package compat

import internalcompat "github.com/vrooli/browser-automation-studio/internal/compat"

// NormalizeWorkflowDefinitionV2Bytes returns protojson-ready bytes for a
// schema-shaped WorkflowDefinitionV2 document.
func NormalizeWorkflowDefinitionV2Bytes(body []byte) ([]byte, error) {
	return internalcompat.NormalizeWorkflowDefinitionV2Bytes(body)
}
