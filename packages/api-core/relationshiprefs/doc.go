// Package relationshiprefs parses Vrooli's documentation relationship references.
//
// Relationship references create auditable edges between documentation,
// implementation, and requirements:
//
//	[CODE: api/internal/server.go#Run]
//	[DOC: docs/reference/api.md#health]
//	[REQ: OT-P0-005]
//
// The package only extracts syntax and normalizes target paths. Callers remain
// responsible for resolving paths, validating anchors, and deciding whether
// broken references are warnings or errors.
package relationshiprefs
