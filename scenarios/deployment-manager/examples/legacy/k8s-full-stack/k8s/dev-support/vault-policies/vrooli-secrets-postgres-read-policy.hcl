# KVv2 Policy: Path must include '/data/' after the engine mount point.
# Example: path "secret/data/vrooli/..."

path "secret/data/vrooli/secrets/postgres" {
  capabilities = ["read", "list"]
} 