# BAS calibration (planted-error fixtures)

Playbooks here are **expected to fail**. They live outside `bas/cases/` so
`test-genie registry build` never registers them into the green suite. Their job
is to prove that a corresponding registered playbook can actually detect a broken
component — otherwise a green run of the real playbook proves nothing.

Calibration fixtures are capability-scoped and are consumed by the owning
experience evaluator. They must name a property, not a component or version;
the generic focus-containment, accessible-name, token-resolution, dark-parity,
and portal-boundary fixtures are the regression boundary for those probes.
