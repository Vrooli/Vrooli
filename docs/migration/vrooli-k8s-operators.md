# Vrooli-Specific K8s Operator Configuration

This file preserves the Vrooli-specific operator requirements that were extracted from:
- `scripts/app/lifecycle/deploy/k8s-prerequisites.sh`
- `scripts/app/lifecycle/deploy/k8s-dependencies-check.sh`

When the vrooli-dashboard scenario is created, this configuration should be moved there.

## Required Operators for Vrooli

```bash
# Operators required by Vrooli platform
declare -A VROOLI_OPERATORS=(
    ["postgresclusters.postgres-operator.crunchydata.com"]="CrunchyData PostgreSQL Operator|postgres-operator|postgres-operator.crunchydata.com/control-plane=postgres-operator"
    ["redisfailovers.databases.spotahome.com"]="Spotahome Redis Operator|redis-operator|app.kubernetes.io/name=redis-operator"
    ["vaultsecrets.secrets.hashicorp.com"]="Vault Secrets Operator|vault|app.kubernetes.io/name=vault-secrets-operator"
)
```

## Vault Configuration Checks

```bash
# Vault connectivity checks (environment-specific)
VAULT_DEV_ADDRESS="http://vault.vault.svc.cluster.local:8200"
VAULT_PROD_ADDRESS="${VAULT_ADDR:-https://vault.example.com:8200}"

# Required Vault configurations:
# - KV secrets engine enabled at "secret/"
# - Kubernetes auth method enabled
# - Proper service account permissions
```

## Health Checks

```bash
# Additional Vrooli-specific health checks:
# - Vault connectivity from vrooli-server deployment
# - Database connectivity via PGO PostgreSQL cluster
# - Redis connectivity via Spotahome Redis failover
```

## Usage Example (for future vrooli-dashboard scenario)

```bash
# In vrooli-dashboard's service.json:
{
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "check-k8s-operators",
          "run": "source scripts/lib/runtimes/k8s/operators.sh && k8s::operators::check_list VROOLI_OPERATORS",
          "description": "Verify required Kubernetes operators"
        }
      ]
    }
  }
}
```