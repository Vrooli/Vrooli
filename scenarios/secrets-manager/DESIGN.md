# Design — Secrets Manager

## Intent

Secrets Manager is an operations console. It must make secret posture, deployment readiness, and remediation actions understandable without exposing secret values. The UI uses dark operational surfaces, semantic status labels, and concise action-oriented copy.

## Feedback & State

Each network-backed panel shows loading, ready, empty, and error states. A status color is always paired with text. Mutations report their result in the owning panel and preserve the server response for operator diagnosis.

## Request Lifecycle

The React UI calls the API through `ui/src/lib/api.ts`. The API is the source of business decisions. UI and CLI surfaces must not duplicate Vault, deployment-strategy, or security-scanning logic.
