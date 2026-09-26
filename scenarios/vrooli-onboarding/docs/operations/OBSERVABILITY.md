# Observability

Use `make logs` for lifecycle-managed logs and the Validation step for operator
readiness. Logs and readiness responses contain credential metadata only. They
must never include values, recovery passphrases, or shell-export material.
