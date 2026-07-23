# Operator control-plane policy. Assign only to the Secrets Manager operator
# identity, never to Prompt Manager's workload identity.

path "transit/keys/prompt-manager-experiment-receipts" {
  capabilities = ["read"]
}

path "transit/keys/prompt-manager-experiment-receipts/rotate" {
  capabilities = ["update"]
}
