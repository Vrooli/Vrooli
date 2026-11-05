# Trusted experiment receipt signing

Prompt Manager protects audit and holdout evidence with a versioned signature
envelope. The envelope records the signing purpose, digest, algorithm, key
identifier, and signature; the receipt content remains durable evidence.

## Production boundary

New production signatures use Ed25519 through the Vrooli credential authority:

```text
vrooli/prompt-manager/experiment-receipts / key-ring
```

The authority chooses native operating-system secure storage or Vrooli's
encrypted backup-controlled store. Prompt Manager and Secrets Manager never
select a backend, read a Vault path, or receive a secret through the process
environment. The key ring contains the active private key plus retained public
verifiers for prior versions. Rotation appends a version rather than deleting
old keys, so already-recorded evidence remains verifiable.

The signer is intentionally split from the API-core contract. API-core owns
the provider-neutral envelope and health interface; the credential-authority
binding owns key-ring encoding, Ed25519 signing, rotation, and local
verification. This avoids making API-core depend on Vrooli's platform-private
credential implementation.

Secrets Manager keeps the operator endpoint because rotation is an operational
action, not an application action. In production the endpoint requires a
verified TLS 1.3 client certificate and an allowlisted Common Name. Once
authorized, it calls the authority signer to bootstrap or append the key ring.
No operator secret is needed beyond the mTLS private key protected by the
deployment's file boundary.

## Compatibility

`vault-transit` remains a supported legacy algorithm in the provider-neutral
package. An optional compatibility wrapper routes only envelopes explicitly
marked with that algorithm to a separately configured legacy Vault verifier;
new `Sign` calls always use the authority signer. With no legacy configuration,
historical Vault envelopes are rejected with a clear fail-closed error rather
than silently treated as authority signatures.

The fixed purposes are `experiment-audit-receipt-v1` and
`experiment-holdout-receipt-v1`. The purpose participates in domain-separated
signing, and production gates require a healthy production signer. Development
HMAC envelopes remain useful for local testing but are never production
eligible.
