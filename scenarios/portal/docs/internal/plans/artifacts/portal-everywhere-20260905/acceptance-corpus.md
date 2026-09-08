# Required acceptance corpus

This is the planning specification, not evidence of passed tests. Implement each case in its owner suite. Map IDs to requirement IDs during phase 2. Each passed case needs a producer receipt, result assertion, source fingerprint, and applicable support row.

| ID | Owner | Behavior |
|---|---|---|
| CAT-01 | portal | Local host identity |
| CAT-02 | portal | Partial inventory |
| CAT-03 | portal | Ambiguous names |
| CAT-04 | portal | Multiple transports |
| CAT-05 | portal | Stale readiness |
| CAT-06 | portal | Headless node |
| CAT-07 | portal | Untrusted descriptor |
| CAT-08 | portal | Attached topology |
| NAT-01 | device-control | Real capture probe |
| NAT-02 | device-control | User-session isolation |
| NAT-03 | device-control | Windows semantic action |
| NAT-04 | device-control | macOS revoke |
| NAT-05 | device-control | X11 geometry |
| NAT-06 | device-control | GNOME Wayland |
| NAT-07 | device-control | KDE Wayland |
| NAT-08 | device-control | Unicode input |
| NAT-09 | device-control | Held input release |
| NAT-10 | device-control | Session lock |
| NAT-11 | device-control | Semantic ambiguity |
| NAT-12 | device-control | Geometry revision |
| NAT-13 | device-control | Helper restart |
| NAT-14 | device-control | Protected control |
| AUTH-01 | device-control | Wrong target |
| AUTH-02 | device-control | Expired grant |
| AUTH-03 | device-control | Concurrent controllers |
| AUTH-04 | device-control | Epoch takeover |
| AUTH-05 | device-control | Duplicate command |
| AUTH-06 | device-control | Conflicting duplicate |
| AUTH-07 | device-control | Crash after effect |
| AUTH-08 | device-control | Observation only |
| AUTH-09 | device-control | Forged helper |
| REM-01 | vrooli-bridge | Terminal parity |
| REM-02 | vrooli-bridge | Protocol confusion |
| REM-03 | vrooli-bridge | Node revocation |
| REM-04 | vrooli-bridge | Loss before effect |
| REM-05 | vrooli-bridge | Loss after effect |
| REM-06 | vrooli-bridge | Slow viewer |
| REM-07 | vrooli-bridge | Relay only |
| REM-08 | vrooli-bridge | Attached revoke |
| OPT-01 | portal | No control providers |
| OPT-02 | portal | No audio |
| OPT-03 | portal | No runtime |
| OPT-04 | portal | Stopped provider |
| OPT-05 | portal | Denied setup |
| OPT-06 | portal | Wrong account fallback |
| OPT-07 | portal | Wrong destination fallback |
| OPT-08 | portal | Optional timeout |
| OPT-09 | portal | Provider recovery |
| OPT-10 | portal | Required profile |
| UI-01 | portal | Presentation parity |
| UI-02 | portal | Full UI reuse |
| UI-03 | portal | Pre-focus capture |
| UI-04 | portal | Shortcut conflict |
| UI-05 | portal | Dismiss focus |
| UI-06 | portal | Annotation scaling |
| UI-07 | portal | Screenless device |
| UI-08 | portal | Keyboard access |
| UI-09 | portal | Display removal |
| UI-10 | portal | Explicit quit |
| UI-11 | portal | Voice finalization |
| UI-12 | portal | Capture denied |
| EMB-01 | portal | Forged origin |
| EMB-02 | portal | Native IPC attempt |
| EMB-03 | portal | Oversized message |
| EMB-04 | portal | Stale surface message |
| EMB-05 | portal | Embed failure |
| EMB-06 | portal | Prompt injection |
| FLOW-01 | device-control | Cross-surface replay |
| FLOW-02 | device-control | Failed candidate |
| FLOW-03 | device-control | Incomplete candidate |
| FLOW-04 | device-control | Weakened repair |
| FLOW-05 | device-control | Stale version |
| FLOW-06 | device-control | Old revision replay |
| FLOW-07 | device-control | Vision budget |
| FLOW-08 | device-control | Demonstration authoring |
| FLOW-09 | device-control | Cross-owner program |
| FLOW-10 | device-control | Undeclared binding |
| PKG-01 | scenario-to-desktop | Vanilla regression |
| PKG-02 | scenario-to-desktop | Unknown extension |
| PKG-03 | scenario-to-desktop | Clean install |
| PKG-04 | scenario-to-desktop | Update continuity |
| PKG-05 | scenario-to-desktop | Interrupted update |
| PKG-06 | scenario-to-desktop | Uninstall policy |
| PKG-07 | scenario-to-desktop | Artifact identity |
| PKG-08 | scenario-to-desktop | Native privilege isolation |
| LEARN-01 | portal | Missing telemetry |
| LEARN-02 | portal | Test provenance |
| LEARN-03 | portal | Comparable reuse |
| LEARN-04 | portal | Unreliable sample |
| LEARN-05 | portal | Assertion integrity |
| MIG-01 | portal | History migration |
| MIG-02 | portal | Capture replacement |
| MIG-03 | portal | Support claims |

## CAT-01 — Local host identity

Owner: `portal`. Required: yes.

```gherkin
Given The companion host differs from the Portal API host
When The user selects This machine
Then The descriptor identifies the companion host
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-02 — Partial inventory

Owner: `portal`. Required: yes.

```gherkin
Given One provider is offline and another is healthy
When The catalog refreshes
Then Healthy targets remain and the failed source is explicit
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-03 — Ambiguous names

Owner: `portal`. Required: yes.

```gherkin
Given Two nodes have the same display name
When A task uses that name
Then No actuation occurs until identity is resolved
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-04 — Multiple transports

Owner: `portal`. Required: yes.

```gherkin
Given A device has ADB and remote-control transports
When Inventory is joined
Then One device exposes distinct transport capabilities
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-05 — Stale readiness

Owner: `portal`. Required: yes.

```gherkin
Given Cached readiness expires
When A task requires control
Then The route probes or reports unknown before action
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-06 — Headless node

Owner: `portal`. Required: yes.

```gherkin
Given A compute node has no desktop session
When The user opens its target
Then Terminal support remains and desktop is unavailable
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-07 — Untrusted descriptor

Owner: `portal`. Required: yes.

```gherkin
Given A provider returns an arbitrary endpoint and token
When The descriptor is projected
Then Secrets and arbitrary connection authority are rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## CAT-08 — Attached topology

Owner: `portal`. Required: yes.

```gherkin
Given A phone is attached to a remote node
When The hierarchy renders
Then The phone and host retain distinct linked identities
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-01 — Real capture probe

Owner: `device-control`. Required: yes.

```gherkin
Given A screenshot executable exists but capture permission is denied
When Readiness is inspected
Then Capture is denied rather than ready
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-02 — User-session isolation

Owner: `device-control`. Required: yes.

```gherkin
Given Two OS users have desktop sessions
When One session receives a control request
Then Only the authorized session can be addressed
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-03 — Windows semantic action

Owner: `device-control`. Required: yes.

```gherkin
Given The Windows fixture exposes an Export button
When A semantic flow invokes it
Then The expected artifact is independently verified
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-04 — macOS revoke

Owner: `device-control`. Required: yes.

```gherkin
Given A capture grant is revoked
When The helper next observes
Then It returns permission evidence without stale success
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-05 — X11 geometry

Owner: `device-control`. Required: yes.

```gherkin
Given The fixture moves to a negative-origin display
When A fresh action resolves
Then The intended control receives input
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-06 — GNOME Wayland

Owner: `device-control`. Required: yes.

```gherkin
Given The portal grants a selected desktop session
When The fixture flow runs
Then Declared capture and input actions verify
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-07 — KDE Wayland

Owner: `device-control`. Required: yes.

```gherkin
Given The selected backend provides its declared portal interfaces
When The fixture flow runs
Then Declared capture and input actions verify
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-08 — Unicode input

Owner: `device-control`. Required: yes.

```gherkin
Given The fixture accepts non-ASCII text
When The flow enters the specified text
Then The exact persisted text matches
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-09 — Held input release

Owner: `device-control`. Required: yes.

```gherkin
Given A drag or modifier is active
When Stop is issued
Then Held input is released and queued input is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-10 — Session lock

Owner: `device-control`. Required: yes.

```gherkin
Given The desktop becomes locked
When A control operation arrives
Then It refuses with an actionable session state
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-11 — Semantic ambiguity

Owner: `device-control`. Required: yes.

```gherkin
Given Two windows expose identical labels
When An underspecified selector resolves
Then It returns ambiguity without clicking
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-12 — Geometry revision

Owner: `device-control`. Required: yes.

```gherkin
Given Display layout changes after observation
When An old coordinate action arrives
Then It is rejected or re-resolved before actuation
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-13 — Helper restart

Owner: `device-control`. Required: yes.

```gherkin
Given A helper restarts after lease issuance
When An old epoch sends input
Then The prior epoch is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## NAT-14 — Protected control

Owner: `device-control`. Required: yes.

```gherkin
Given An elevated or protected OS surface is inaccessible
When The agent requests interaction
Then The adapter reports unsupported or denied without false success
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-01 — Wrong target

Owner: `device-control`. Required: yes.

```gherkin
Given A grant names node A
When Input names node B
Then No native action occurs
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-02 — Expired grant

Owner: `device-control`. Required: yes.

```gherkin
Given A grant has expired
When Input arrives
Then The destination refuses
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-03 — Concurrent controllers

Owner: `device-control`. Required: yes.

```gherkin
Given Two actors request control
When Both attempt actuation
Then Only the current lease holder acts
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-04 — Epoch takeover

Owner: `device-control`. Required: yes.

```gherkin
Given A user takes control
When A queued agent input arrives
Then The stale epoch is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-05 — Duplicate command

Owner: `device-control`. Required: yes.

```gherkin
Given A completed command is submitted again unchanged
When The owner reconciles
Then It returns the original receipt without another effect
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-06 — Conflicting duplicate

Owner: `device-control`. Required: yes.

```gherkin
Given A command ID is reused with changed payload
When The owner admits it
Then It refuses a conflict
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-07 — Crash after effect

Owner: `device-control`. Required: yes.

```gherkin
Given The helper crashes after native effect but before final receipt
When The client reconnects
Then The outcome remains unknown until independently reconciled
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-08 — Observation only

Owner: `device-control`. Required: yes.

```gherkin
Given A session has only view permission
When Input is requested
Then The destination refuses
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## AUTH-09 — Forged helper

Owner: `device-control`. Required: yes.

```gherkin
Given An unauthenticated process claims a user session
When Registration is attempted
Then The owner rejects it
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-01 — Terminal parity

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given An existing PTY session is running
When Desktop support is enabled
Then Terminal byte and resize semantics remain correct
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-02 — Protocol confusion

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given A terminal frame is sent to a desktop session
When Admission evaluates it
Then It rejects the wrong protocol
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-03 — Node revocation

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given Media and input channels are active
When The node is revoked
Then Existing channels lose authority
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-04 — Loss before effect

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given The route disconnects before command admission
When The client reconnects
Then The owner can safely resume the unperformed step
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-05 — Loss after effect

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given The route disconnects after submission
When An alternate provider is available
Then The original receipt is reconciled before fallback
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-06 — Slow viewer

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given A viewer cannot consume frames
When The stream continues
Then Queues remain bounded and stop remains responsive
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-07 — Relay only

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given Direct peer connectivity is unavailable
When A supported relay route is selected
Then Authorized viewing works with recorded cost and latency
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## REM-08 — Attached revoke

Owner: `vrooli-bridge`. Required: yes.

```gherkin
Given A device grant is revoked while the host stays online
When Device input arrives
Then Device control stops without revoking unrelated host sessions
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-01 — No control providers

Owner: `portal`. Required: yes.

```gherkin
Given Client profile has no BAS or Device Control
When Portal starts
Then Text UI and capability explanations work
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-02 — No audio

Owner: `portal`. Required: yes.

```gherkin
Given Audio Tools is absent
When The user opens the composer
Then Text entry works and voice has a clear unavailable state
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-03 — No runtime

Owner: `portal`. Required: yes.

```gherkin
Given Program Runtime is absent
When The user requests a composed task
Then The task reports its prerequisite without a private executor
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-04 — Stopped provider

Owner: `portal`. Required: yes.

```gherkin
Given A required provider is installed but stopped
When An authorized recovery is selected
Then The owner lifecycle starts it and readiness is rechecked
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-05 — Denied setup

Owner: `portal`. Required: yes.

```gherkin
Given Provider installation lacks authority
When A task needs that provider
Then Portal requests missing authority without installing
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-06 — Wrong account fallback

Owner: `portal`. Required: yes.

```gherkin
Given Remote BAS uses a different account
When A local task loses its provider
Then The route is rejected as inequivalent
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-07 — Wrong destination fallback

Owner: `portal`. Required: yes.

```gherkin
Given A task explicitly targets Office PC
When Local control is available
Then The router does not redirect silently
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-08 — Optional timeout

Owner: `portal`. Required: yes.

```gherkin
Given A provider readiness call hangs
When The deadline expires
Then Other Portal actions remain usable
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-09 — Provider recovery

Owner: `portal`. Required: yes.

```gherkin
Given An absent provider becomes available
When The user retries the retained task
Then The same task context resolves a fresh valid route
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## OPT-10 — Required profile

Owner: `portal`. Required: yes.

```gherkin
Given A profile requires Device Control
When Neither bundle nor configured endpoint supplies it
Then Setup fails explicitly before claiming readiness
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-01 — Presentation parity

Owner: `portal`. Required: yes.

```gherkin
Given A conversation has attachments and a running task
When The user changes pill palette and expanded modes
Then Conversation branch target and run identity survive
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-02 — Full UI reuse

Owner: `portal`. Required: yes.

```gherkin
Given The companion expands
When The ordinary workspace mounts
Then The same Portal components and contracts serve it
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-03 — Pre-focus capture

Owner: `portal`. Required: yes.

```gherkin
Given Another app owns focus
When The global shortcut opens Portal
Then Context refers to the prior app
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-04 — Shortcut conflict

Owner: `portal`. Required: yes.

```gherkin
Given The desired hotkey is unavailable
When The companion starts
Then It reports conflict and offers reconfiguration
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-05 — Dismiss focus

Owner: `portal`. Required: yes.

```gherkin
Given The palette is dismissed
When The prior app remains available
Then Focus returns according to the documented platform behavior
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-06 — Annotation scaling

Owner: `portal`. Required: yes.

```gherkin
Given A region is marked on a scaled display
When The context opens in the full UI
Then The annotation maps to the original source geometry
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-07 — Screenless device

Owner: `portal`. Required: yes.

```gherkin
Given A transport supports buttons and properties only
When Its surface opens
Then A useful control panel appears without a fake screen
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-08 — Keyboard access

Owner: `portal`. Required: yes.

```gherkin
Given The user cannot use a pointer
When They navigate surfaces and stop a task
Then All essential actions are keyboard accessible
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-09 — Display removal

Owner: `portal`. Required: yes.

```gherkin
Given The pill is on a removed monitor
When Display topology changes
Then The companion remains reachable on an available display
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-10 — Explicit quit

Owner: `portal`. Required: yes.

```gherkin
Given A task is active
When The user quits
Then The selected continue-or-stop disposition is honored
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-11 — Voice finalization

Owner: `portal`. Required: yes.

```gherkin
Given Duplicate final transcript events arrive
When Submission occurs
Then Only one task is created
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## UI-12 — Capture denied

Owner: `portal`. Required: yes.

```gherkin
Given OS capture permission is denied
When The user invokes Portal
Then Text chat remains usable with a clear context limitation
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-01 — Forged origin

Owner: `portal`. Required: yes.

```gherkin
Given A hostile iframe sends a context message
When The host validates it
Then The message is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-02 — Native IPC attempt

Owner: `portal`. Required: yes.

```gherkin
Given Embedded content invokes a native shell method
When The trusted boundary evaluates it
Then No native operation occurs
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-03 — Oversized message

Owner: `portal`. Required: yes.

```gherkin
Given A child sends an oversized payload
When The host receives it
Then It rejects boundedly without blocking chat
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-04 — Stale surface message

Owner: `portal`. Required: yes.

```gherkin
Given A closed surface sends input
When The host checks session identity
Then The message is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-05 — Embed failure

Owner: `portal`. Required: yes.

```gherkin
Given One scenario iframe crashes
When The user continues chat
Then The workspace remains usable
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## EMB-06 — Prompt injection

Owner: `portal`. Required: yes.

```gherkin
Given An app displays instructions to exfiltrate context
When The agent observes it
Then Task authority remains unchanged
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-01 — Cross-surface replay

Owner: `device-control`. Required: yes.

```gherkin
Given A saved desktop flow version exists
When Browser Portal companion and CLI run it
Then All use the same owner execution and assertions
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-02 — Failed candidate

Owner: `device-control`. Required: yes.

```gherkin
Given A candidate execution fails
When Promotion is requested
Then No saved version is created
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-03 — Incomplete candidate

Owner: `device-control`. Required: yes.

```gherkin
Given A run has no final outcome evidence
When Promotion is requested
Then The owner refuses
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-04 — Weakened repair

Owner: `device-control`. Required: yes.

```gherkin
Given A repair removes an assertion
When Validation runs
Then The candidate is rejected
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-05 — Stale version

Owner: `device-control`. Required: yes.

```gherkin
Given A repair names an old expected version
When It attempts persistence
Then The owner returns a conflict
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-06 — Old revision replay

Owner: `device-control`. Required: yes.

```gherkin
Given A newer version has been promoted
When An exact old version is selected
Then The original immutable procedure executes
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-07 — Vision budget

Owner: `device-control`. Required: yes.

```gherkin
Given An unfamiliar task exhausts its budget
When The agent requests another step
Then Execution terminates with a bounded unresolved result
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-08 — Demonstration authoring

Owner: `device-control`. Required: yes.

```gherkin
Given A user demonstrates a successful task
When The system creates a candidate
Then Promotion still requires assertions and replay
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-09 — Cross-owner program

Owner: `device-control`. Required: yes.

```gherkin
Given A task combines browser export and desktop verification
When The program runs
Then Each owner supplies its own attributable receipt
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## FLOW-10 — Undeclared binding

Owner: `device-control`. Required: yes.

```gherkin
Given Generated code requests an undeclared operation
When Runtime validates it
Then It rejects execution before the operation
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-01 — Vanilla regression

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A non-Portal vanilla consumer is generated
When The extension mechanism is present
Then The ordinary package still builds and runs
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-02 — Unknown extension

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A package declares an unsupported extension version
When Generation starts
Then It fails with a typed compatibility reason
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-03 — Clean install

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A primary host has no prior Portal installation
When The candidate installs
Then Its declared profile opens with correct readiness
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-04 — Update continuity

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A supported predecessor contains user state
When The candidate updates it
Then Conversation state and helper identity are preserved
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-05 — Interrupted update

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given An update is interrupted
When The app recovers
Then It runs a coherent version or reports recovery without corrupt state
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-06 — Uninstall policy

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given The user chooses the documented retention option
When Uninstall runs
Then Only owned artifacts are removed accordingly
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-07 — Artifact identity

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A candidate was tested
When Release readiness is evaluated
Then Its digest matches the tested bytes
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## PKG-08 — Native privilege isolation

Owner: `scenario-to-desktop`. Required: yes.

```gherkin
Given A package loads remote scenario content
When The content attempts host access
Then Sandbox and IPC policy deny it
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## LEARN-01 — Missing telemetry

Owner: `portal`. Required: yes.

```gherkin
Given An effort field was not measured
When A report is generated
Then It remains absent rather than zero
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## LEARN-02 — Test provenance

Owner: `portal`. Required: yes.

```gherkin
Given Synthetic corpus attempts exist
When Operator learning is measured
Then Test attempts are excluded
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## LEARN-03 — Comparable reuse

Owner: `portal`. Required: yes.

```gherkin
Given Fresh and reused tasks share acceptance and context
When Effort is compared
Then Denominators and failures remain visible
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## LEARN-04 — Unreliable sample

Owner: `portal`. Required: yes.

```gherkin
Given The sample is capped or too small
When Setpoints are read
Then The report marks unreliable instead of in band
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## LEARN-05 — Assertion integrity

Owner: `portal`. Required: yes.

```gherkin
Given A speed optimization reduces checks
When The corpus runs
Then The regression is detected instead of counted as improvement
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## MIG-01 — History migration

Owner: `portal`. Required: yes.

```gherkin
Given Old Assistant has representative issue records
When Migration runs on a copy
Then Counts identifiers and links reconcile
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## MIG-02 — Capture replacement

Owner: `portal`. Required: yes.

```gherkin
Given The user invokes old issue-capture intent
When Portal captures context and routes it
Then A current owner task receives the evidence once
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.

## MIG-03 — Support claims

Owner: `portal`. Required: yes.

```gherkin
Given A platform row is unverified
When Launch material is reviewed
Then It does not claim support for that row
```

Evidence: producer run/receipt reference, assertion result, source fingerprint, and environment/support row.
