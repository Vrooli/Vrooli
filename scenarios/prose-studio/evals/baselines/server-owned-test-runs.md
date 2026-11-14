# Durable validation is a product capability

Vrooli's test-genie owns a scenario test run after the command returns. That matters when a suite takes longer than an agent session: the run keeps its identity, logs, and result instead of disappearing with the caller.

The workflow is simple. Start the scenario suite, record the run identifier, and wait once on the server-owned run. A cancelled client is not the same thing as an aborted run. This gives operators durable evidence for long-running validation and makes a later handoff possible.

The distinction is operational, not cosmetic. A system that treats every disconnected terminal as a failed test loses the difference between infrastructure interruption and a product defect. Server-owned runs preserve that distinction so the next operator can inspect the evidence and make a grounded decision.

The result is a control plane that can improve itself without asking every agent to stay attached to a terminal. Durable runs turn waiting into a governed capability: the work continues, the evidence remains addressable, and the final decision can be made from the same record that produced it.
