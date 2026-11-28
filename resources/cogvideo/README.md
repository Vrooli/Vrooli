# Local video generation resource

This resource boundary is reserved for a locally owned video model/runtime that can
serve a maintained consumer through a durable execution receipt. The initial
scaffold is not an installed or qualified service. It must not be started or used
to claim video generation until a complete, reproducible runtime and model profile
passes resource validation, live host admission, and a real generation proof.

## Intended contract

- Keep the model card, exact upstream revision, model files, license obligations,
  supported task capabilities, runtime, and device/memory limits together as data.
- Expose a bounded local job lifecycle (submit, inspect, wait, cancel) and retain
  durable execution identity, selected model revision, seed, outputs, and measured
  resource/latency facts.
- Keep model files and generated outputs in separate resource storage entries.
  Acquisition and cleanup must use the resource owner; output cleanup must never
  remove unique evaluation evidence.
- Preserve the Asset Studio → AI Gateway → durable receipt consumer as the target
  integration. AI Gateway video role/executor/catalog work belongs to LM4 and is
  currently closed; this resource does not make that route live.

## Candidate status — 2026-11-16

`zai-org/CogVideoX-2b` at
`1137dacfc2c9c012bed6a0793f4ecf2ca8e7ba01` has a pinned Apache-2.0 license file
with SHA-256
`4c3f6aeb71e12415a59117dfd35cfc177514424f0356c81798fde1a239d0b6a8`. Its
model files and required configuration total 13,775,572,738 bytes at that
revision. The license copy, modification notices, and applicable upstream notices
must ship with any redistribution. Model metadata, its 4-GB memory claim, and
imports on the host are not a hardware or quality qualification.

At the latest host sample, the filesystem had 139 GiB free, leaving 19 GiB above
the required 120-GiB reserve. Since the resource dependency closure is not yet
isolated and the complete install/run peak has not been measured, this candidate
is not admitted. No model bytes, runtime install, inference, or render have run.
The resource remains an unfinished owner path pending a smaller complete profile
or a verified runtime/data peak within the 20-GB trial limit, a fresh capacity
admission, and owner-level quality/performance evidence.

## Operations

The generated CLI currently exposes only standard lifecycle commands. Do not
acquire data or start the service: `resource.json` is still scaffold metadata and
does not yet define a complete executable runtime, acquisition group, or model
selection policy. Follow `docs/resources/authoring-a-resource.md`, use governed
resource acquisition and the Scenario Dependency Analyzer for dependency closure,
and preserve the shared host capacity and disk admission boundaries.
