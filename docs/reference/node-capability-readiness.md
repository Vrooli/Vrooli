# Node capability readiness

## Contract

Vrooli MUST report two independent readiness axes for a terminal target:

- transport readiness: whether the control plane can reach and authorize the target;
- capability readiness: whether the target can run a named coding agent.

A missing capability MUST NOT make a reachable target undispatchable. An empty
shell remains available when no coding agent is installed.

Each capability observation MUST use one of these states:

| State | Meaning | Operator action |
| --- | --- | --- |
| `ready` | The command is on PATH and its version was read. | Launch the capability. |
| `missing` | The command is not on PATH. | Install the named resource or use another target. |
| `not_applicable` | No acquisition route exists for this platform. | Use the vendor's manual route or another target. |
| `unknown` | The target has not reported a fresh observation. | Refresh the probe and check node reporting. |

The node agent owns observation. Bridge owns transport and durable inventory
storage. Web Console owns projection and presentation. The launcher MUST read
the selected target's capability facts and MUST refuse a missing, unsupported,
or unknown agent before opening a PTY.

## Platform applicability

A required item with state `not_applicable` MUST NOT block onboarding. A missing
or unknown required item MUST remain visible as a blocker. A malformed manifest
or invalid acquisition declaration MUST block because it is a contract defect,
not a platform limitation.

## External CLI acquisition

An `external-cli` resource with an `install-direct` command MUST declare an
`acquisition` contract. Each supported target MUST identify its platform and a
SHA-256 digest. Unsupported platforms MUST use an explicit `unsupported`
target. The fleet contract check MUST reject a new external CLI that bypasses
this declaration.

## Per-machine configuration

A machine profile is the desired configuration document. It includes scenarios,
optional resources, required capabilities, and setup environment. The machine
record stores the applied profile id, version, and time. Drift is a typed list
of differing items computed from desired profile, applied record, and current
capability inventory; it MUST NOT be represented only by a boolean or prose.

An onboarding operation that pairs without applying configuration MUST use the
`paired` terminal state. An operation that applies configuration MUST use the
configured success state. Every terminal operation MUST record its operation id,
terminal state, completion time, and names of items not applied on the node.

