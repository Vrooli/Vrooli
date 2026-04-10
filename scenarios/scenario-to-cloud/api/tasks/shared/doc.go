// Package shared provides reusable context attachment builders for agent tasks.
//
// This package contains pure functions that construct context attachments used
// by both investigation and fix task handlers. Each builder produces a single
// ContextAttachment that can be included in agent prompts.
//
// Design principles:
//   - Pure functions: deterministic output for same input
//   - No side effects: no I/O, no state mutation
//   - Composable: builders can be combined freely
//   - Testable: easy to unit test in isolation
package shared
