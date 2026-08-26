# Internal control plane

The `internal/` tree is the Vrooli control plane. Packages here own host
observation and remediation policy, scenario lifecycle, credentials, operator
state, and the application and CLI boundaries that expose those capabilities.
Browse it by package family and ownership boundary rather than treating the
directory listing as a hierarchy.

## Package families

| Family | Packages | Read in this order |
| --- | --- | --- |
| Host requirements and host state | `hostreqspec`, `hostreqkit`, `hostreq`, `hostreqcheck`, `hostreqrun`, `hostsession`, `hostfacts`, `hostinventory`, `hostpressure`, `hostpresentation`, `hostlifecycle` | specification → shared inspection/apply kit → requirement model and checks → execution/session → facts and inventory → pressure/presentation → lifecycle |
| Credentials | `credentialpolicy`, `credentialauthority`, `credentialspec`, `credentialinventory`, `credentialescrow`, `credentials` | policy → authority → specification → inventory → escrow → keyring and repair operations |
| Scenarios | `scenario`, `scenarioruntime`, `scenarioenv`, `scenarioexec`, `scenariostale` | model → runtime → environment → execution → stale-process observation |
| Operator state and capability | `operatorcapability`, `operatorinput`, `operatorstate` | capability contract → input collection → durable state |

The adjacent `hostcapability`, `gpuaccess`, `accel`, `capabilitycatalog`, and
`deployability` packages provide capability observations and resolution. The
`artifact*`, `acquisition`, `portspec`, `daemonreload`, `localprincipal`, and
`structureprovider` packages provide shared control-plane boundaries. Resource
and safeguard-specific packages remain close to the handlers they serve.

## CLI layers

CLI command surfaces use three layers when an application boundary earns its
own package:

```text
internal/app/<family>       service or application orchestration
internal/cli/<family>cli     command registration and CLI-facing types
internal/cli/<family>handlers handler wiring and command behavior
```

The established small-family shape covers authentication, capacity, contract,
hygiene, package, project, recovery, resource, and scenario commands. New
families should follow the same shape when the application layer contains real
orchestration. A thin pass-through package is not a reason to add a layer; the
larger resource and safeguard subsystems may bind directly to their CLI layer
when that is the honest ownership boundary.

## App-layer responsibility

The `internal/app/` layer exists when domain logic is shared by more than one
consumer or when a command family needs a transport-free orchestration
boundary. A package with no consumer is removed. A thin package with one
consumer is retained only when it owns a real service contract; otherwise its
caller may bind directly to the domain package. Existing `resourcecli`,
`scenariocli`, and `projectcli` direct-domain bindings remain intentional.

## Why the tree stays flat

Compound package names such as `hostinventory` and `credentialinventory` keep
the family and responsibility visible at every call site. Naively nesting
these packages and shortening their names creates six known collisions:

1. `scenarioruntime` becomes `runtime`, colliding with the standard library,
   `internal/runtime`, and `internal/resources/runtime`.
2. `scenarioexec` becomes `exec`, colliding with `os/exec`.
3. `hostinventory` and `credentialinventory` both become `inventory`.
4. `hostlifecycle` becomes `lifecycle`, colliding with `internal/lifecycle`.
5. `scenarioenv` becomes `env`, colliding with `internal/resources/env`.
6. `credentials` has no useful short leaf name after nesting.

The full measurements, alternatives, and decision record are in the [internal
consolidation survey](../docs/architecture/internal-consolidation-survey.html).
Keeping the packages flat preserves readable import names and avoids an
import-alias migration whose only benefit would be directory tidiness.

`internal/resources/` is the intentional existing exception: its nested
subsystems (`env`, `runtime`, `manifest`, and `control`) are imported through
compound aliases at call sites. That history is a warning against repeating
the pattern for the cross-cutting host, credential, and scenario families.
