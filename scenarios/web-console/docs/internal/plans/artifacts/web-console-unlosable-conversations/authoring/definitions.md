Conversation catalog — The Web Console record that maps pane identity, agent identity, transcript sources, titles, lifecycle state, and recovery lineage.
Durable evidence — Conversation events, native agent history, checkpoints, lineage, or metadata required to find and resume prior work.
Lifecycle receipt — The immutable result of one typed lifecycle command, keyed by a stable operation identifier.
Metadata orphan — Durable evidence whose canonical catalog or session metadata record is absent or inconsistent.
Local continuity search — Web Console search for its own sessions and recovery evidence across every lifecycle state.
Global conversation search — Agent Manager retrieval across runs, harnesses, and conversation sources, federated by Search Hub.
Reconciliation generation — A fingerprinted inventory used to prove that an apply operation acts on the reviewed evidence set.
Repair tombstone — A preserved catalog record that represents historical evidence without claiming a live process exists.
Native transcript — Agent-runtime-owned history such as a Codex rollout; Web Console reads but does not rewrite it.

