# Trusted experiment receipt signing

Prompt Manager protects audit and holdout evidence with a versioned signature
envelope. The envelope records the signing purpose, digest, algorithm, key
identifier, and signature; the receipt content remains durable evidence.

Production uses the non-exportable Vault Transit key
`prompt-manager-experiment-receipts`. Prompt Manager receives a short-lived
workload credential through a permission-restricted lifecycle credential file,
not a raw signing key, Vault root token, Vault KV value, SQLite field, or custom
scenario environment variable. Its policy permits only Transit sign, verify,
and metadata-read operations for that key.

The fixed purposes are `experiment-audit-receipt-v1` and
`experiment-holdout-receipt-v1`. A signature from one purpose cannot verify as
the other. The canonical receipt bytes are SHA-256 digested before signing.

The lifecycle declaration in Prompt Manager's `.vrooli/service.json` contains
endpoint, key name, and credential-file metadata only. The standard runtime
config path remains a compatibility fallback for old local checkouts. The
explicit development declaration uses a labelled development signer; its
receipts are never production eligible. Historical signature-only HMAC receipts
are likewise rejected by the trusted verification gate.

Vault key rotation retains previous key versions so existing envelopes continue
to verify. The operator-facing status must expose only provider health and key
identity, never key material or the workload credential.
