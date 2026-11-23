# Model evaluation phase

The AI Gateway owns the Test Genie `model-eval` phase identity. Its provider
reads the Ollama model-policy roles, the separate AI Gateway inference task-role
catalog, every distinct consumer scenario observed in the Gateway route ledger,
the complete ledger's selected effective runtime model identities, and Search
Hub's registered suites, fixed-denominator snapshots, and current provider-owned
corpus-validation receipts bound to the exact suite and denominator IDs. It counts
only reviewed caller-audit cases with linked Agent Manager source, Gateway
route/request, consumer scenario, capacity, and independent label evidence.
Policy roles, task roles, consumers, and observed runtime models are separately
reported. Policy roles, task roles, and consumers are independent coverage denominators. The
phase reports suite IDs and role/consumer counts without returning query or
prompt contents.

The readiness provider does not run inference or mutate suites. Missing or
invalid snapshots or validation receipts, underfilled policy roles, gateway task
roles or consumers, and the live-inference admission state are returned as
explicit blocking evidence. Corpus validation reports stale expected documents
without deleting them; the fixed-denominator grader keeps them in the recall
denominator as misses. A ready evidence inventory still waits for the separate
operator-routed LM2 inference admission before a model-evaluation run can execute.

When Test Genie requests execution for this phase, AI Gateway delegates to the
Agent Manager owner endpoint. Agent Manager reads the current answer for the
exact LM2 decision, freezes up to 20 authenticated operator user-message
identities, re-reads each message before its sequential Gateway call, and
returns metadata only. It rejects a reused approval, changed scope, remote
route, denied or unknown rights, missing privacy proof, stale capacity, or
insufficient disk/artifact headroom. The production prompt/output privacy
verifier is intentionally fail-closed until transport, provider, and logging
sinks have a complete deployed non-persistence proof. A refused execution is
retained as a Test Genie terminal receipt and does not count as a model result.
