# REST exceptions

Every onboarding route not listed here must be a generated Connect procedure.
The transport drift gate rejects a retained REST route without a matching
entry and one of the four canonical `RESTReason` values.

| Method | Path | RESTReason | Justification | Owner |
|---|---|---|---|---|
| GET | `/health` | `ops_probe` | Lifecycle probes, load balancers, and curl need a dependency free operational probe. | onboarding health module |
| GET | `/api/v1/health` | `ops_probe` | Client facing health alias remains reachable to lifecycle and load balancer integrations without a generated client. | onboarding health module |
