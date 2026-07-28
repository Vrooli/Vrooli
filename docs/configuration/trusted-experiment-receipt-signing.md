# Configure trusted experiment receipt signing

Production deployments must provision a Vault Transit key named
`prompt-manager-experiment-receipts`, apply
`resources/vault/policies/prompt-manager-experiment-receipts.hcl` to the Prompt
Manager workload identity, and write a 0600 credential file owned by that
workload. Vault must be served over TLS.

Assign `resources/vault/policies/secrets-manager-experiment-receipts-operator.hcl`
only to the separate Secrets Manager operator identity. It is the sole identity
allowed to rotate the Transit key; Prompt Manager never receives that capability.
Secrets Manager exposes `POST /api/v1/vault/receipt-signing/rotate`, but it
fails closed unless the request has a verified mTLS client certificate whose
Common Name appears in its lifecycle declaration's `operator_subjects`. The
Secrets Manager deployment uses its distinct `operator_credential_file` for
the Vault request. A request header, body field, or an application workload
credential cannot authorize rotation.

The lifecycle declaration in Prompt Manager's `.vrooli/service.json` carries
this metadata (production deployment replaces the development declaration):

```json
{
  "trust_signing": {
    "provider": "vault-transit",
    "resource": "vault",
    "address": "https://vault.example.internal",
    "key_name": "prompt-manager-experiment-receipts",
    "credential_file": "/run/vrooli/identities/prompt-manager-vault-token"
  }
}
```

This declaration deliberately contains no Vault token or signing key. Prompt Manager
fails closed for production promotion if the configuration, TLS connection,
credential file, Transit key, policy, or signature verification is unavailable.

Secrets Manager uses a separate production declaration for the rotation control.
The lifecycle supplies the operator Vault credential and the mTLS files through
protected files. These fields name file locations only. They never contain a
certificate private key, a Vault token, or Transit key material.

```json
{
  "trust_signing": {
    "provider": "vault-transit",
    "resource": "vault",
    "address": "https://vault.example.internal",
    "key_name": "prompt-manager-experiment-receipts",
    "credential_file": "/run/vrooli/identities/secrets-manager-vault-health-token",
    "operator_credential_file": "/run/vrooli/identities/secrets-manager-vault-rotate-token",
    "operator_subjects": ["matthalloran8"],
    "operator_tls_cert_file": "/run/vrooli/tls/secrets-manager.crt",
    "operator_tls_key_file": "/run/vrooli/tls/secrets-manager.key",
    "operator_tls_client_ca_file": "/run/vrooli/tls/operator-client-ca.pem"
  }
}
```

When this declaration selects `vault-transit`, Secrets Manager starts its API
with TLS 1.3 and requires a verified client certificate for every request. Do
not terminate TLS before the Secrets Manager process for the rotation route.
The process must receive the verified client TLS connection so it can enforce
the declared `operator_subjects` allowlist. A development declaration starts
the ordinary HTTP API, but its rotation route remains unavailable.
