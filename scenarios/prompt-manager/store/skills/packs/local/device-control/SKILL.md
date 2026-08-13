# Agent skill: drive a physical Android phone

Use the `device-control` CLI as the agent boundary. Never address an ADB
serial directly from an automation plan; first resolve a physical device id
from `device-control device list --json`.

## Safe sequence

1. Run `device-control device connect --kind android --json` and stop if the
   live rung is unavailable. Follow its named next action for missing `adb`, a
   charge-only cable, an offline device, missing permissions, or an
   unauthorized RSA prompt.
2. Run `device-control device list --json`. Select a row with `kind=physical`,
   `health=available`, and the intended serial/model.
3. Acquire a lease: `device-control session acquire --device <id> --actor <actor>`.
4. Validate a snake_case flow against `android-adb` before execution.
5. Run it with `device-control flow run --device <id> --actor <actor> --lease
   <lease-token> --file <flow.json>`. Keep the lease token; the run reuses and
   does not release an operator-owned lease.
6. Review chapter disposition, retained evidence references, applied redaction
   rules, and `device-control audit list`.
7. Release the lease, or use `device-control session kill <id>` when the run
   must stop immediately.

Flows should prefer semantic targets and use coordinates only when the target
cannot be resolved from the Android accessibility tree. Evidence is redacted
by default. An unredacted capture requires an actor and creates an explicit
audit exception.