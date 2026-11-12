# Model evaluation phase

The AI Gateway owns the Test Genie `model-eval` phase identity. Its provider
reads the Ollama model-policy roles, the separate AI Gateway inference task-role
catalog, and Search Hub's registered suites and fixed-denominator snapshots.
It counts only reviewed caller-audit cases with linked Agent Manager source,
Gateway route/request, capacity, and independent label evidence. The phase
reports suite IDs and role counts without returning query or prompt contents.

The readiness provider does not run inference or mutate suites. Missing or
invalid snapshots, underfilled policy roles or gateway task roles, and the
live-inference admission state are returned as explicit blocking evidence. A
ready evidence inventory still waits for the separate operator-routed LM2
inference admission before a model-evaluation run can execute.
