# Caller replay

`ai-gateway routing replay` defaults to bounded route preview over a JSONL set.
Preview validates caller provenance and asks the gateway policy path for a route
decision; it does not send prompt text to a model provider. Use `--mode execute`
only after model execution has its own capacity admission.

Each line contains one caller-shaped text-generation request:

```json
{"case_id":"case-001","source_run_id":"11111111-1111-4111-8111-111111111111","source_event_id":"22222222-2222-4222-8222-222222222222","source_sequence":3,"conversation_id":"conversation-id","role":"chat.default","profile":"local-only","privacy_class":"internal","operation":"chat.complete","scenario":"agent-manager","timeout_ms":120000,"required_vram_bytes":1073741824,"capacity_estimate_evidence_uri":"capacity-footprint://ollama/dynamic","input_text":"the original caller message"}
```

The run, event and sequence identifiers must refer to an Agent Manager message
event. Before any route preview or inference is dispatched, the command reads
each exact event through Agent Manager's `GetRunEvents` API, then checks the run
ID, event ID, sequence, user role, and exact message content. The message's
conversation ID is checked when supplied. Pass `--agent-manager-url` or set
`AGENT_MANAGER_API_URL`; it must point to a loopback URL, and the CLI forwards
its authenticated caller identity. A corpus author must still independently
review the target policy role and the case's label before admitting it to an
acceptance suite. The command authenticates the source message, not its quality
label.

Run a reviewed corpus with bounded concurrency:

```sh
ai-gateway routing replay --input-file caller-cases.jsonl --agent-manager-url http://localhost:18800 --parallel 4 --receipt-file replay-receipt.json
```

Use `--mode execute` to opt into local inference after the source set and
capacity are admitted. Execute mode requires each case to declare a VRAM
reservation from 1 byte through 16 GiB and a `capacity-footprint://` evidence
URI identifying its owner-measured sizing basis; AI Gateway must return a
matching `fit` claim with granted bytes at least equal to that reservation.
`preview` is the default and does not require a VRAM reservation.
The URI is retained as provenance, but the replay command does not resolve or
authenticate it; the corpus owner must verify that the referenced owner
footprint record supports the requested byte estimate before accepting the case.

The runner accepts at most 220 cases, 64 KiB of input text per case, a 10 MiB
JSONL file, and concurrency from 1 to 16. It requires local-only routing and
internal privacy by default; paid or remote inference is not admitted. Preview
receipts contain source references, decision status, selected provider, route
plan, and route-preview duration. Execute receipts additionally bind the
verified Agent Manager run/event/sequence to the persisted Gateway `event_id`
and `request_id`, plus a `gateway_role_evidence_uri` for the route's role
assignment. It also records the caller-requested VRAM bytes, the capacity-estimate
evidence URI, claim ID, verdict, broker-required bytes, and granted bytes; the
command refuses to mark an execution captured if route
identity or matching capacity admission is absent. They also contain selected
model, locality, and execution latency. Both modes omit prompts and model
output. Execution errors use a stable code to avoid copying provider error text
into receipts.

The bound route is a new replay of that exact user message under the role in the
input file; it does not prove the original conversation historically selected
that role. Independent role/label review is still required before acceptance.
This path currently grades route availability only. Preview duration is not
model latency and must not be reported as p95 inference. Neither mode does
qualify answer correctness or source-to-role/label mapping, produce Search Hub
retrieval denominators, or replace the Test Genie `model-eval` phase. Do not
count generated or unreviewed cases toward an acceptance suite.
