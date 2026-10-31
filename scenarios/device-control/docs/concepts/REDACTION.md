# Capture redaction policy

Authorized local capture follows the existing device lease, evidence access,
audit, and retention behavior. Images and recordings are unredacted by default.
Set `redact_sensitive_content: true` on a flow to request the configured
producer transform; the Device Detail capture panel exposes the same choice as
an unchecked option.

The current image transform masks the status-bar band and any explicit flow
regions. Video masks only the status-bar band. Text transforms recognize the
configured credential and six-digit-code patterns. These transforms do not
semantically classify the rest of a screen or recording.

`redaction_verified` means only that a requested transform completed. With no
redaction request, metadata records `redaction_verified: false`,
`redaction_not_requested`, and `content_classification_not_performed`. A failed
requested transform is withheld. See
[`redaction-policy.md`](redaction-policy.md) for the full rule and metadata
contract.

Local capture authority does not authorize external sharing or publication.
Those paths retain their separate approval, classification, and retention
rules.
