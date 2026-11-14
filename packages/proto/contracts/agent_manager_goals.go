// Package contracts contains shared limits for API fields whose callers span
// multiple scenarios. Keep these values with the wire-contract module so every
// producer and consumer validates the same budget without a new dependency.
package contracts

// AgentManagerMaxGoalCompletionBytes bounds Agent Manager's engine-owned
// completion condition. This is a transport/storage safety ceiling, not a
// prompt-size target; callers should keep goals concise and put durable design
// in docs.
const AgentManagerMaxGoalCompletionBytes = 1 << 20

// AgentManagerMaxGoalPromptBytes bounds a composed goal task prompt. It is
// independent from the completion condition because they are separate fields.
const AgentManagerMaxGoalPromptBytes = 1 << 20
