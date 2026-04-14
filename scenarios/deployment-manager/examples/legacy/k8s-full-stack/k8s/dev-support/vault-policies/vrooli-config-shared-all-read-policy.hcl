# KVv2 Policy: Path must include '/data/' after the engine mount point.
# Example: path "secret/data/vrooli/..."

path "secret/data/vrooli/config/shared-all" {
  capabilities = ["read", "list"]
}
path "secret/data/vrooli/config/shared-all/*" { // In case of sub-paths
  capabilities = ["read", "list"]
} 