# Evidence Status

| Status | Meaning |
|---|---|
| `proven` | Evidence supports the requested fact or proof. |
| `missing` | Expected evidence was not found. |
| `contradicted` | Evidence conflicts with a declaration or requested proof. |
| `unsupported` | Code Facts or a provider cannot evaluate this target/family. |
| `unknown` | Evaluation was attempted but inconclusive, usually due to provider failure or partial graph data. |

Every status should carry a message, analyzer/provider when relevant, and source location when available.
