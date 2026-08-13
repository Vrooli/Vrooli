# Device Control capture redaction policy

Device Control treats every screen capture as private data. Redaction is
performed by the device-control producer before an evidence reference is
published or an artifact is retained. Consumers receive a reference, checksum,
size, timestamp, producer, and applied-rule list; they do not receive a raw
filesystem path or an implicit claim that a capture was safe.

## Default rules

This policy is default-deny: a region is private and must be redacted unless
the policy explicitly names it safe for the claim being produced. A detector
may add sensitive regions, but no consumer may remove a region or turn a
failed verification into a warning. An unknown media format is refused.

The default policy applies these named rules:

- `status_bar_identifiers`: mask the status-bar band, which can contain the
  device owner's carrier, account, and notification identifiers.
- `notification_content`: mask the notification region, because notifications
  can contain private message content, codes, and calendar data.
- `credential_patterns`: redact credential-shaped text in text-based captures,
  including authorization headers, bearer tokens, API keys, secrets, and
  passwords.
- `one_time_codes`: redact six-digit one-time-code-shaped text.
- `flow_sensitive_regions`: mask any pixel regions explicitly marked sensitive
  by the flow author.

An evidence reference records the rules that were applied. If an image cannot
be decoded and safely transformed, the run fails rather than publishing an
unverified capture. Redaction verification is computed from the completed
producer operation; it is never a hard-coded success value.

## Audited opt-out

An owner or operator may explicitly set `allow_unredacted_capture` on a flow
only for a controlled debugging or review session. The request must name the
actor. A request without an actor is rejected before the flow starts. The
resulting evidence reference is marked `opted_out`, and the audit record names
the actor and records the opt-out as an exception.

Unredacted captures may be viewed only by the owner/operator who authorized
the opt-out and holds the device lease. Downstream consumers cannot disable
redaction after the producer boundary. Retention and access controls still
apply to the resulting artifact.
