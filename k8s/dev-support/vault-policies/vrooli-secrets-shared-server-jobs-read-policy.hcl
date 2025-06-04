# KVv2 Policy: Path must include '/data/' after the engine mount point.
# Example: path "secret/data/vrooli/..."

path "secret/data/vrooli/secrets/shared-server-jobs" {
  capabilities = ["read", "list"]
}
path "secret/data/vrooli/secrets/shared-server-jobs/*" { // In case of sub-paths
  capabilities = ["read", "list"]
} 