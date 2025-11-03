package metadata

const (
	AgentLastErrorKey       = "agent_last_error"
	AgentFailureTimeKey     = "agent_failure_time"
	AgentCancelReasonKey    = "agent_cancel_reason"
	AgentTranscriptPathKey  = "agent_transcript_path"
	AgentLastMessagePathKey = "agent_last_message_path"
	AgentSessionIDKey       = "agent_session_id"
	AgentProviderKey        = "agent_provider"
	RateLimitUntilKey       = "rate_limit_until"
	RateLimitAgentKey       = "rate_limit_agent"
	PreferredAgentKey       = "preferred_agent"
	BlockedByIssuesKey      = "blocked_by_issues"   // CSV of issue IDs causing target conflict
	BlockedAtKey            = "blocked_at"          // RFC3339 timestamp when block was detected
	BlockedReasonKey        = "blocked_reason"      // Reason code: "target_conflict", "rate_limited", etc.

	AgentStatusExtraKey          = "agent_last_status"
	AgentStatusTimestampExtraKey = "agent_last_status_at"
)
