// Package releasedecision persists the pending operator decisions raised when
// an agent asks to ship a release it is not yet authorized for ("ask, don't
// block"). It implements operatordecision.PendingStore over a SQLite table so
// a raise is idempotent, flood-capped, and resolvable by the notification-hub
// ask-resolution webhook. The durable approval effect itself stays in the
// readiness/approvals domains; this package only owns the pending-ask ledger.
package releasedecision

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the pending release-decision ask
// ledger. It is registered in the boot-time schema set alongside the other
// deployment-manager module schemas.
func Schema() string { return schemaSQL }
