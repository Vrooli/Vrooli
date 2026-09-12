# Unit Health adapter contract

Unit Health adapters are versioned capabilities. The kernel selects an adapter
from observed Code Facts, declared policy, and host support; it never infers a
runner from a language alone.

An adapter should provide:

- immutable `id` and semantic `version` identity;
- a match predicate covering language, observed framework, and platform;
- typed test and coverage commands (`executable`, `argv`, environment,
  working directory, timeout, resources, and artifacts);
- bounded artifact readers that return normalized coverage/evidence records;
- optional policy-projection and architecture analyzers;
- explicit missing-tool, unsupported-host, degraded-fact, and malformed-output
  diagnostics; and
- conformance fixtures for passing, failing, missing-runtime, timeout, and
  malformed-artifact cases.

Commands must contain an executable and argument vector. Adapters must not
evaluate shell strings or install dependencies. Dependency provisioning and
readiness belong to Scenario Dependency Analyzer; Unit Health only consumes
the governed readiness result. Resolve launchers through the host-resolution
seam before producing a plan, preserve the resolved absolute path (including
paths with spaces and platform executable extensions), and emit a typed
missing-tool diagnostic when resolution fails.

Artifact kinds are adapter-owned identifiers (for example Go cover profiles,
Istanbul summaries, LCOV, and Cobertura XML). A planner declares the relative
artifact paths it produces, and the adapter reads only those declared kinds.
The validation kernel consumes the normalized record and does not dispatch on
language or framework names.

Policy projection settings are intentionally opaque in the generic testing
schema. An adapter validates its own settings and may reject unknown or
malformed keys. New adapters should add their own conformance tests and
register an analyzer only when they can prove the complete contract on the
claimed host platforms.
