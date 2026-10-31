# Device Control capture redaction policy

Device Control captures screens only through the existing device lease and
owner evidence flow. For authorized local use, capture is unredacted by default.
This matches ordinary personal-device research and debugging workflows.
Evidence remains subject to the existing local storage, artifact access, audit,
and retention behavior.

## Optional redaction

A flow can set `redact_sensitive_content: true` to request the configured
producer transform. The default is `false`; a capture with no transform records
`redaction_verified: false`, `redaction_not_requested`, and
`content_classification_not_performed`.

For images, the current transform masks the status-bar band when enabled and any
sensitive regions explicitly supplied by the flow. It does not scan or classify
the rest of the screen. Text captures apply configured credential and
six-digit-code patterns. Video captures mask the status-bar band. Unsupported
formats or a failed requested transform are refused.

`redaction_verified: true` means only that the declared transform completed. It
does not mean the content was semantically classified or found free of personal
information. Evidence metadata reports the applied rules and this limitation.

The legacy `allow_unredacted_capture` field remains accepted for compatibility
and retains its actor audit. New local captures do not need an opt-out because
unredacted capture is the normal default.

## Sharing and retention

Local evidence access does not authorize external sharing or publication. Those
operations remain governed by their own approval, classification, and retention
rules. This policy does not change evidence storage or retention boundaries.
