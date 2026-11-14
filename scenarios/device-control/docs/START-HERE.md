# Start Here — Device Control

Device Control drives owner-authorized TVs, Google TVs, Android phones and
tablets, and other devices through leased, audited, evidence-backed operations.
Read the section that matches what you are doing.

## I want a device to do something

Use the one-call verbs. Each wakes the screen when needed, verifies its own
result, and on failure prints the exact next command (`next_actions`).

| Ask | Command |
|---|---|
| Which devices can I use? | `device-control device list` (`--all` adds unreachable rows, `--full` every probe) |
| Screenshot of the TV or phone | `device-control device screenshot "SmartTV 4K" [--home]` |
| Short screen recording | `device-control device record "<name>" --seconds 8` |
| Open Netflix, YouTube, Settings, any package | `device-control device open "<name>" --app netflix` |
| Wake the screen or go Home | `device-control device screen "<name>" --operation ready [--home]` |
| What is on the screen now | `device-control device state "<name>"` or `semantics` |
| Save evidence again | `device-control device evidence <evidence-id> --out file` |
| Same asks as programs (with learning) | `program-runtime library run device-control.check-app` (screenshot, or install an APK, open it, capture it) |
| Change volume | `program-runtime library run device-control.volume` |
| Reuse or author a multi-step flow | `device-control.do-task`, `author-flow`, `replay-flow` |

A TV known only over Google Cast has no screen frame. Cast, the Android TV
Remote and ADB-over-network are three transports of one physical TV; the
capture verbs need ADB. When a capture ends in `capability_gap`, run the command
it names: usually `device relations` then `device relation-confirm <candidate>`,
or `device onboard-network --endpoint <tv-ip>:5555 --attach-to <tv-id>` when no
ADB identity exists. The TV needs Developer options, Network debugging, and a
first "Allow USB debugging" tap on its screen.

A locked phone unlocks itself when it has an unlock profile: `device-control auth list` shows it, and the setup is
`device-control auth create --device <id> --method pin --credential-identity device-control/<id>/unlock --credential-field pin`
then `read -rs PIN && printf %s "$PIN" | device-control auth provision <profile-id>`. The PIN never appears in a command line or
in chat, and a profile whose last attempt failed is not retried until re-provisioned.

## I want to understand it

- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md): identity, transports, leases, flows.
- [reference/capabilities.md](reference/capabilities.md): what each modality means and how it is probed.
- [guides/connecting-a-device.md](guides/connecting-a-device.md): phone, Google TV and ADB setup.
- [internal/DECISIONS.md](internal/DECISIONS.md): choices with alternatives and switch costs.

Core rules: capabilities are probed, never inferred from device kind; a
capability gap is a successful, honest answer with its next step; one physical
device has one durable identity however many transports reach it; every effect
holds a lease and writes an audit record.

## I want to change it

- Start a scenario: `make start`; logs: `make logs`; stop: `make stop`.
- Tests: `vrooli scenario test device-control --phases <phase>`; see
  [internal/TESTING.md](internal/TESTING.md). Specifications of behavior live as
  Gherkin `.feature` files under `api/internal/control/features/`, run by
  `TestFeatures` through `api/internal/testutil/gherkin`.
- API/CLI/UI share proto contracts under `packages/proto/schemas/device-control`.
  Regenerate only through `cd packages/proto && make generate SCENARIO=device-control`.
- The CLI surface is declared in `cli/manifest.json`; program bindings derive from it.
- Record a non-obvious choice in [internal/DECISIONS.md](internal/DECISIONS.md) and
  known gaps in [internal/PROBLEMS.md](internal/PROBLEMS.md).
