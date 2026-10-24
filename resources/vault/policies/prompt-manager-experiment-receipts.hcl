# Least-privilege policy for Prompt Manager experiment receipt signing.
# The workload may sign and verify only this non-exportable Transit key. It
# cannot read key material, access Vault KV, create keys, or rotate keys.

path "transit/sign/prompt-manager-experiment-receipts" {
  capabilities = ["update"]
}

path "transit/verify/prompt-manager-experiment-receipts" {
  capabilities = ["update"]
}

path "transit/keys/prompt-manager-experiment-receipts" {
  capabilities = ["read"]
}
