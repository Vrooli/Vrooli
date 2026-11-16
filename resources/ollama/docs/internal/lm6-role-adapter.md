# LM6 Ollama role adapter: owner boundary map

The Ollama adapter owns installed-model identity, Ollama policy eligibility, capability matching, and the local-inference rights preflight. It does not own model-evaluation evidence, optimizer disk admission, promotion, or cross-consumer deletion.

## Existing owner operations

| Concern | Existing owner boundary | What it proves | LM6 status |
|---|---|---|---|
| Installed candidates | Managed Ollama loopback `/api/tags`, through the resource `ensure.Client` inventory | Installed reference, content digest, and size | Read-only inspection is wired by `resource-ollama optimize role-adapter` |
| Role fit and rights | `resources/ollama/model-policy.json` and `policy.CheckLocalInferenceRights` | Default eligibility, rights record and local-inference decision for the candidate | Adapter checks required target-role capabilities and records the policy digest; candidate rights preflight runs against the policy owner |
| Acquisition | `resource-ollama ensure` and `internal/ensure.DiskAdmission` | Governed pull and a shared capacity disk-job lease for the complete pull peak | Existing models can be inspected; missing models must use `ensure`; optimizer request is not yet connected to this operation |
| Evaluation | AI Gateway owns Test Genie `model-eval` readiness; live execution is separately admitted by LM2 | Readiness inventory only; no candidate score or evaluation receipt | No execution API is available. A supplied receipt ID is opaque and is never accepted as evaluation evidence |
| Disk/capacity | Shared `packages/capacity` broker, used by Ollama ensure for model pulls | Owner-issued disk-job lease and receipts for pull work | No optimizer evaluation/promotion admission operation currently exists |
| Promotion | No resource-owned role-policy transaction is currently exposed | — | Refused; policy inspection is read-only |
| Candidate cleanup | `ensure.Client.DeleteModel` removes a model tag through Ollama | Tag removal only; it does not prove absence of consumer or rollback references | Refused; no complete cross-consumer reference proof exists |

## Callable inspection contract

`resource-ollama optimize role-adapter --request <json-file> --json` accepts
`ollama.optimize.role-request.v1` and returns `ollama.optimize.receipt.v1`.
The adapter accepts any role present in the authoritative model policy, with three
unique candidates, a fixed suite and at least 20 unique case IDs. LM6's showcase
remains specifically three `chat.default` candidates over 20 or more fixed cases;
the same contract can also carry canonical narrow roles for downstream LM8 work.
The caller cannot provide capabilities or rights metadata: strict JSON decoding
rejects unknown fields, and the adapter reads role requirements and model rights
from the single Ollama policy owner. Unknown and blank roles fail closed. The
read-only inspection request does not require `evaluation_receipt_id`; the engine
has not run its Evaluate stage yet. If a caller includes an opaque ID, inspection
still does not verify it. `EvaluationVerified` remains false, and the receipt
carries an evaluation-unverified refusal. A future operation that verifies
evaluation must require a non-empty receipt issued after evaluation.
The output reports each candidate's current source-rights decision and digest,
trusted context identity/version/digest, evaluated timestamp, exact policy digest,
and source artifact-rights identity. This rights decision is evaluated even when
the model is not installed. Runtime inventory digest, size and role capability
gaps remain separate installed-artifact observations. It always remains `refused`
while evaluation verification,
optimizer capacity admission, restart-safe promotion, or cleanup proof is absent.

The adapter is an inspection owner operation; it is not a Program Runtime binding.
Program Runtime lives in a separate Go module, and the engine's Adapter interface
is internal to that scenario. Calling this CLI through a private shell execution
would bypass the governed binding boundary. Until an owner-backed callable boundary
is added and registered, do not claim that the optimizer invoked the role adapter.

## Source-only constraints

This contract may be validated with Go unit tests. Running ensure, inference, model
acquisition, service lifecycle actions, GPU work, or capacity-sensitive tests
requires a fresh `/home` footprint check and owner admission under LM6's 120 GiB
free-space floor.
