# Capture redaction policy

Device Control treats every screen as sensitive until the producer proves
otherwise. The default producer policy masks password fields, one-time codes,
authorization headers, API keys, secrets, and credential-shaped text before a
capture becomes an evidence reference. The reference contains only checksum,
size, creation time, recording provenance, content-verification status, and the
verified-redaction bit; it never contains frame bytes or a filesystem path.

An unredacted capture is an explicit flow-level opt-out. It must name an actor,
be recorded in the audit trail, and is available only to the owner/operator
who holds the device lease. Consumers cannot turn redaction off after the
producer boundary. If a detector cannot verify the policy, the capture is
rejected rather than emitted with an ambiguous status.

Synthesized recordings carry `method: synthesized` and an effective frame
rate. Rates below 5 FPS are degraded and cannot support a release-grade claim.
Native recordings are also decoded and sampled outside the status/navigation
bands before publication. A valid container with a uniformly near-black body
is rejected as failed evidence rather than being presented as a successful
recording.
`ios-mirror` is structurally `advisory-ocr` and non-promotable for release
evidence even when its transport is available.
