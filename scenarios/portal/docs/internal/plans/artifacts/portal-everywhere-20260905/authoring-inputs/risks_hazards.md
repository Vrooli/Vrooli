| Risk | Concrete mitigation | Evidence |
|---|---|---|
| Wrong-machine input | Bind target and desktop session in every control grant. | Mismatched-target adversarial tests. |
| Stale screen geometry | Attach display ID, scale, origin, rotation, and observation revision. | Move/resize/DPI tests. |
| Competing human and agent | Shared target-side lease and control epoch. | Takeover race tests. |
| Duplicate side effect after disconnect | Durable command identity and outcome-unknown reconciliation. | Disconnect-after-actuation tests. |
| Optional dependency cascade | Lazy adapters and bounded readiness probes. | Missing-provider package matrix. |
| Native IPC privilege leak | Typed allowlisted methods and sender verification. | Hostile iframe/renderer tests. |
| Full-screen private content disclosure | Selected context, redaction, bounded retention, explicit policy. | Artifact access and cleanup tests. |
| Workflows overfit one app version | Compatibility constraints and independent outcome assertions. | Version/layout corpus. |
| Single-platform implementation drift | Early cross-platform primitive suite. | Per-platform live receipts. |
| Forked packaging template | Versioned extension contract. | Regeneration and vanilla-consumer tests. |
| Marketing outruns support | Claims-to-evidence mapping. | Release review. |
| Self-improvement optimizes an easy subset | Fixed corpus and denominator reporting. | Comparable-window benchmark receipts. |

Implementation stakes justify repairing understood blocking contract, test-provider, packaging, or transport defects. They do not justify unrelated feature expansion. Record T1/T2 changes through Plan Manager. Obtain a plan revision for changes to the promised outcome.
