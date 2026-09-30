// Package vroolierr owns typed control-plane error classification and presentation helpers. It does not decide domain remediation.
//
// Codes are stable, lower-snake-case failure identities. Use usage_error for
// malformed operator input, environment_error for missing host prerequisites,
// runtime_error for failed operations, internal_error for uncategorized HTTP
// failures, and untyped_cli_error only as the CLI boundary fallback. Domain
// errors should use a more specific code such as scenario_not_found.
//
// Categories are presentation classes: Usage, Environment, Runtime, and
// Internal. A code identifies a failure; a category only controls how it is
// shown to an operator.
package vroolierr
