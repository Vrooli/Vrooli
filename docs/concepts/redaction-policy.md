# Device-control capture redaction policy

Device Control is the producer and custodian of captured bytes. Captures are
redacted before they become evidence references. The default policy masks
status-bar identifiers and credential-shaped, password, and one-time-code
content whenever the adapter provides those regions or text metadata. The
reference records the checksum, size, creation time, producer, and rules that
actually ran; it never exposes bytes or a filesystem path through the API.

Only an owner or an operator acting under an active device lease may request an
unredacted capture. The request must identify the actor. Device Control writes
that actor, the opt-out, and the computed redaction result to the audit record.
Consumers cannot disable redaction after the producer boundary. If a capture
cannot be safely redacted, it is rejected rather than marked verified.
