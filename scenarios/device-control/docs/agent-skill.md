# Agent skill: drive a physical Android phone

Use the `device-control` CLI as the agent boundary. Never address an ADB
serial directly from an automation plan; first resolve a physical device id
from `device-control device list --json`.

## Safe sequence

1. Run `device-control device connect --kind android --json` and stop if the
   live rung is unavailable. Follow its named next action for missing `adb`, a
   charge-only cable, an offline device, missing permissions, or an
   unauthorized RSA prompt.
   For a previously promoted wireless device, use
   `device-control device reconnect <device-id> --json` when the saved endpoint
   is stale; reconnect verifies the original hardware serial before persisting
   a discovered TLS endpoint and never enables wireless debugging for you.
2. Run `device-control device list --json`. Select a row with `kind=physical`,
   `health=available`, and the intended serial/model.
3. Acquire a lease: `device-control session acquire --device <id> --actor <actor>`.
4. Inspect, create, or update a reference-only profile with
   `device-control auth list` / `device-control auth create` /
   `device-control auth update`; provision its value only by piping stdin to
   `device-control auth provision <profile-id>`. Never put a credential in a
   flow, flag, environment variable, or JSON file. Delete temporary authority
   values with `device-control auth delete-credential` before revocation.
5. If the state report says `lock_state=locked`, run
   `device-control auth unlock --profile <profile-id> --device <id> --lease
   <lease-token> --json` and require `outcome=unlocked` plus
   `after_lock_state=unlocked` before continuing. `human_required`, provider
   failures, unknown state, and wrong credentials are terminal for the flow.
6. Validate a snake_case flow against `android-adb` before execution.
7. Run it with `device-control flow run --device <id> --actor <actor> --lease
   <lease-token> --file <flow.json>`. Add `--transport wireless` explicitly
   for a promoted wireless device. Keep the lease token; the run reuses and
   does not release an operator-owned lease.
8. Review chapter disposition, retained evidence references, applied redaction
   rules, and `device-control audit list`.
9. Release the lease, or use `device-control session kill <id>` when the run
   must stop immediately.

Flows should prefer semantic targets and use coordinates only when the target
cannot be resolved from the Android accessibility tree. Evidence is redacted
by default. An unredacted capture requires an actor and creates an explicit
audit exception.
