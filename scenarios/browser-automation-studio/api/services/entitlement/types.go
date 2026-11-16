package entitlement

import "github.com/vrooli/vrooli/packages/monetization-go"

// Plan and feature vocabulary is shared by all paid-surface consumers.
const (
	TierFree     = string(monetization.PlanFree)
	TierSolo     = string(monetization.PlanSolo)
	TierPro      = string(monetization.PlanPro)
	TierStudio   = string(monetization.PlanStudio)
	TierBusiness = string(monetization.PlanBusiness)
)

type Status = monetization.SubscriptionStatus

const (
	StatusActive   = monetization.StatusActive
	StatusTrialing = monetization.StatusTrialing
	StatusPastDue  = monetization.StatusPastDue
	StatusCanceled = monetization.StatusCanceled
	StatusInactive = monetization.StatusInactive
)

const (
	FeatureAI            = monetization.FeatureAI
	FeatureRecording     = monetization.FeatureRecording
	FeatureWatermarkFree = monetization.FeatureWatermarkFree
)

// Entitlement is kept as a source-compatible alias while BAS handlers move to
// the shared paid-surface snapshot. Authorization still uses the shared Gate.
type Entitlement = monetization.Snapshot
