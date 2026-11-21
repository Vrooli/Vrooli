// Package tokenaccounting owns the vocabulary and arithmetic for Agent
// Manager's token attribution model.
//
// Token factors are deliberately kept separate from their derived views. A
// footprint describes the payload added by one tool call, residency describes
// the weighted turns that carried that payload, and incurred describes usage
// reported by the provider for a turn. The package also provides the explicit
// residual used when a run cannot be completely attributed.
package tokenaccounting
