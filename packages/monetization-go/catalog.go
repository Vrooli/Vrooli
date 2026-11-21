package monetization

// PlanTier and SubscriptionStatus are shared display/catalog vocabulary. They
// are not an authorization boundary: access decisions must use the verified
// lease's PlanRank and status through Gate.
type PlanTier string

const (
	PlanFree     PlanTier = "free"
	PlanSolo     PlanTier = "solo"
	PlanPro      PlanTier = "pro"
	PlanStudio   PlanTier = "studio"
	PlanBusiness PlanTier = "business"
)

type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusTrialing SubscriptionStatus = "trialing"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusInactive SubscriptionStatus = "inactive"
)

const (
	FeatureAI            = "ai"
	FeatureRecording     = "recording"
	FeatureWatermarkFree = "watermark_free"
)
