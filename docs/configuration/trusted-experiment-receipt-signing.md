# Configure trusted experiment receipt signing

Production Prompt Manager receipt signing uses Vrooli's credential authority,
not Vault. The authority selects the host's native secure storage service and
falls back only to Vrooli's encrypted, operator-controlled credential store.
Prompt Manager does not start, discover, or receive credentials from Vault.

The authority stores a versioned Ed25519 key ring at this identity and field:

```text
identity: vrooli/prompt-manager/experiment-receipts
field:    key-ring
```

The private keys remain authority values. The signer stores the public key next
to each private key in the protected ring so rotation can retain historical
verifiers. The key-ring value must be provisioned through the credential
authority or by the Secrets Manager rotation endpoint; it must never be placed
in a scenario file, environment variable, or database.

The production lifecycle declaration contains routing metadata only:

```json
{
  "trust_signing": {
    "provider": "credential-authority-ed25519",
    "identity": "vrooli/prompt-manager/experiment-receipts",
    "field": "key-ring"
  }
}
```

Prompt Manager's checked-in declaration remains `development` for local work.
Development envelopes are deliberately ineligible for production conclusions.
An operator should rotate/bootstrap the production key ring before enabling the
production declaration. Secrets Manager's rotation endpoint remains protected
by TLS 1.3 mutual authentication and its declared `operator_subjects`; mTLS
authorizes the operator, while the credential authority performs the write.
There is no operator Vault credential. The rotation routes are
`POST /api/v1/credentials/receipt-signing/rotate` and
`GET /api/v1/credentials/receipt-signing/status`.

```json
{
  "trust_signing": {
    "provider": "credential-authority-ed25519",
    "identity": "vrooli/prompt-manager/experiment-receipts",
    "field": "key-ring",
    "operator_subjects": ["operator-common-name"],
    "operator_tls_cert_file": "/run/vrooli/tls/secrets-manager.crt",
    "operator_tls_key_file": "/run/vrooli/tls/secrets-manager.key",
    "operator_tls_client_ca_file": "/run/vrooli/tls/operator-client-ca.pem"
  }
}
```

The fixed purposes are `experiment-audit-receipt-v1` and
`experiment-holdout-receipt-v1`. The canonical receipt bytes are SHA-256
digested before signing, and the purpose is included in the Ed25519 signing
domain so a signature cannot be replayed as another receipt type.

The old Vault Transit implementation remains only as an explicit compatibility
verifier for historical envelopes. It is not an active dependency or signing
provider. If historical Vault envelopes must still be checked, a deployment
may add `legacyVaultTransit` to the authority runtime config with a
permission-restricted legacy credential file. Without that explicit optional
configuration, old Vault envelopes fail closed while all new authority
envelopes remain locally verifiable.

Status responses expose provider health and the active key ID only; they never
return a private key, public key-ring value, authority credential, or Vault
token.
