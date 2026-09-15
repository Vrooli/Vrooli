// Package assistantmigration is the API module's compatibility path. The
// implementation is shared with Portal CLI so inventory and export cannot
// drift between operator entry points.
package assistantmigration

import shared "github.com/vrooli/vrooli/scenarios/portal/assistantmigration"

type (
	Record         = shared.Record
	Manifest       = shared.Manifest
	Issue          = shared.Issue
	EvidenceLink   = shared.EvidenceLink
	CaptureRequest = shared.CaptureRequest
	ReviewManifest = shared.ReviewManifest
	CaptureReceipt = shared.CaptureReceipt
	CaptureRouter  = shared.CaptureRouter
)

var (
	Inventory   = shared.Inventory
	Export      = shared.Export
	Reconcile   = shared.Reconcile
	ParseIssue  = shared.ParseIssue
	BuildReview = shared.BuildReview
	Route       = shared.Route
	ErrConflict = shared.ErrConflict
)
