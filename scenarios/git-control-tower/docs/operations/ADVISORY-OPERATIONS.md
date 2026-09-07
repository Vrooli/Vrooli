# Advisory operations

## Startup and health

Start and stop Git Control Tower through the Vrooli lifecycle. Health means the
API process and database are reachable; it does not mean Workspace Sandbox,
Search Hub, Integration Hub, Test Genie, or host publication are available.
The UI must display each dependency's capability standing independently.

## Durable jobs and recovery

Review and mention rows are SQLite-owned by their domains. A request key is
idempotent. On restart, queued/running rows are rehydrated and provider
execution identities remain attached. A failed requested check remains a
failed check; a newer unrelated provider run cannot replace it. Retention must
not remove active jobs or evidence referenced by a result.

## Evidence and privacy

Change subjects and evidence bundles are bounded by repository identity,
revision/digest, scope, coverage, and omissions. `available=false` is distinct
from an available empty result. Public mention output must omit private work
references and internal artifact paths. A provenance path match is not
committed authorship.

## Shutdown and backup

Before shutdown, allow the lifecycle owner to finish its server-owned operation
or leave a durable running/interrupted row. SQLite backups must include the
domain tables and be restored into a disposable validation instance before
claiming recovery. Provider caches and Search Hub indexes are projections and
must be rebuilt from owner evidence.

## Known external activation gaps

Live Integration Hub host credentials, provider publication receipts, and
Workspace Sandbox committed-content evidence are not fabricated by GCT. The
activation packet must show configured, probed, scoped, and human-authorized
standing separately.
