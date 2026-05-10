# Security

## Data Sensitivity

Knowledge records and scenario documentation may contain operational details.
Treat search results and documentation previews as internal by default.

## Auth And Authorization

Current local scenario workflows rely on Vrooli local-stack access controls.
External deployment must add explicit auth boundaries before exposing search or
healing operations.

## Secrets

Configuration should reference resource URLs and credentials through environment
or service configuration, not checked-in documentation.

## Security Gaps

Document-healing approval flows must continue to prevent silent code changes
outside documentation paths.
