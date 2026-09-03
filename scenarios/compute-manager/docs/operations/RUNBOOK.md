# Runbook: Compute Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

> **Status: nothing in this runbook has been built or exercised.**
> Compute Manager was generated from the `react-vite` template on
> 2026-09-03 and contains template code only. There is no provider
> adapter, no instance record, no reconciler, no expiry sweeper and no
> enrollment path. Every procedure below is the procedure that *will*
> apply once the matching requirement is implemented. Each one names the
> requirement that owns it. Treat every command shown as an intended
> command surface, not an available one. Do not treat a green result
> from any step here as evidence until the requirement it belongs to is
> implemented and its automated validations pass.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

Two facts shape everything here, and they are worth stating before the
procedures. First, this scenario spends real money on an hourly clock, so
an incident that would merely be untidy elsewhere is a growing bill here.
Second, the scenario deliberately reports rather than resolves: the
reconciler never deletes anything it finds, because a reconciler bug that
deletes is indistinguishable from a reconciler bug that destroys a paying
customer's node. That design choice is what makes the operator procedures
below necessary, and it is why the first one is a quarantine rather than
an automatic cleanup.

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

Stopping this scenario is safe with respect to cost, by design. An
instance carries a first-boot timer that powers it off at its expiry
without contacting the control plane, so a stopped Compute Manager drains
the fleet rather than stranding it (`COMPUTEM-P0-004`). That timer is not
implemented yet, so at present a stopped Compute Manager would strand
nothing only because it can create nothing.

## Operator Procedures

These are the procedures a human operator performs. They fall into two
groups.

The first three are the manual validation for a P0 requirement and are
referenced by `requirements/01-must-ship/module.json`. All three are
currently recorded as `not_implemented`.

The remaining four are incident responses rather than manual
validations. Each one answers an alert declared in
[`OBSERVABILITY.md`](OBSERVABILITY.md#alerts--health) or a credential
obligation declared in `.vrooli/service.json`, and none of them is
referenced by the requirements registry, because a response to an alert
is not evidence that a requirement holds.

Every command shown below is a verb declared in
[`../reference/cli-commands.md`](../reference/cli-commands.md). If a
procedure here needs a verb that document does not declare, the fix is to
settle the verb there first, not to invent it here.

### Quarantine An Unaccounted Instance

**Owns:** `COMPUTEM-P0-003` (bidirectional reconciliation).
**Status: not implemented, never exercised.**

An unaccounted instance is one the reconciler found at the provider with
no matching local record. It is billing now. It is also the exact shape a
reconciler bug produces, so the procedure inspects before anything is
destroyed, and nothing in the reconciler destroys at all.

Perform these steps:

1. Run `compute-manager reconcile findings --json`.
2. Find the finding with direction `provider-only`.
3. Record the provider, the region and the provider-side identifier.
4. Run `compute-manager reconcile quarantine <finding-id>`.
5. Confirm the finding state is `quarantined`.
6. Run `compute-manager intent list --provider-id <provider-side-id>`.
7. Read the result.
   - If an intent exists, the instance is ours and the local record was
     lost. Go to step 8.
   - If no intent exists, the instance was not created by this scenario.
     Go to step 11.
8. Run `compute-manager instance reclaim <finding-id> --intent <intent-id>`.
9. Confirm the instance now appears in `compute-manager instance list`
   with an elapsed cost and a remaining lifetime.
10. Stop. The instance is metered and expiring. If it should not exist,
    destroy it the ordinary way, with
    `compute-manager instance destroy <instance-id>`, which confirms
    against the instance's own name. This procedure ends here.
11. Confirm with the operator that the instance is not owned by another
    system that shares the provider account.
12. Destroy it wherever it was created. This scenario has no verb that
    destroys a machine it holds no record of, so the provider console,
    the provider CLI, or the owning system is the only path.
13. Run `compute-manager reconcile run` and confirm the finding closes
    because the instance is no longer present at the provider.

Notes on the rationale. Step 4 marks and does not sweep. Quarantine
changes no provider state; it only stops the finding from being reported
again on every sweep, so the queue stays readable. There is deliberately
no `reconcile destroy`: the reconciler reports and never resolves, so a
reconciler defect cannot delete a running node, including a paying
customer's. Destruction runs through the instance surface in step 10 or
outside this scenario in step 12, and the split at step 7 is what decides
which. Reclaim in step 8 is the ownership branch and it starts the meter,
which is why it is a different verb from `enroll adopt`; see the note
under [First Unattended Enrollment](#first-unattended-enrollment).
Provider billing data is not part of this procedure at any step: it lags
by hours to more than a day, which makes it a reconciliation signal and
never a control.

### Verify The Instance-Side Drain

**Owns:** `COMPUTEM-P0-004` (expiry enforced twice).
**Status: not implemented, never exercised.**

Expiry is enforced in two independent places: a sweeper in this scenario,
and a timer written into the instance's own first-boot configuration. The
second exists because the first cannot run when this scenario is down,
and an instance that outlives an unavailable control plane bills forever.
This procedure proves the second one works by removing the first.

Perform these steps:

1. Create an instance with a short lifetime:
   `compute-manager instance request --lifetime 20m --purpose drain-drill`.
2. Wait for the instance state to become `running`.
3. Record the instance identifier and the expiry timestamp.
4. Run `make stop` in this scenario directory.
5. Confirm the API no longer responds on `API_PORT`.
6. Wait until the recorded expiry timestamp has passed by five minutes.
7. Query the provider console or provider CLI directly for the instance.
8. Confirm the provider reports the instance as powered off.
9. Run `make start`.
10. Run `compute-manager reconcile run`.
11. Confirm a finding reports the instance as stopped at the provider.
12. Run `compute-manager instance destroy <instance-id>` to release the
    resource and settle its usage.

Notes on the rationale. Step 8 checks for powered off, not destroyed. The
first-boot timer can only stop the instance it runs on; it cannot delete
the instance, because deletion needs a provider credential and no
credential is ever placed on an instance. Powering off does not stop
billing on most providers, so the drain is a floor and not a fix: it
bounds the damage from an unavailable control plane, and step 12 is what
actually stops the meter. Do not run this drill with a lifetime shorter
than the provider's minimum billable unit, because Hetzner rounds a
partial hour up to a full hour and a short drill costs a full hour either
way.

### First Unattended Enrollment

**Owns:** `COMPUTEM-P0-005` (unattended enrollment).
**Status: not implemented, and blocked upstream.**

Enrollment delegates entirely to `vrooli-bridge`. This scenario contains
no SSH implementation and never holds a node credential. The instance
boots already trusting the bridge onboarding public key, so there is no
interactive step and no secret crosses any wire.

**Blocker:** the bridge does not publish its onboarding public key on any
endpoint. It can read the key internally, but exposes it nowhere.
Publishing it is the single new wire contract this scenario needs, and it
is an upstream prerequisite rather than integration work. This procedure
cannot be performed until that endpoint exists.

Perform these steps:

1. Run `compute-manager status --json`.
2. Confirm the bridge dependency reports healthy and reports the
   onboarding public key as retrievable. There is no separate key
   command; onboarding-key retrievability is a dependency fact and
   belongs to `status`.
3. Run `compute-manager instance request --lifetime 2h --enroll true`.
4. Do not open a terminal to the instance. Do not type a password.
5. Wait for the instance state to become `running`.
6. Run `compute-manager enroll status <instance-id>`.
7. Confirm the bridge Machine record exists and names the instance
   address as its locator.
8. Run `vrooli-bridge machine list --json`.
9. Confirm the machine reports the online state.
10. Search the scenario logs for the instance identifier:
    `make logs | grep <instance-id>`.
11. Confirm no key material, password or token value appears in any line.
12. Run `compute-manager instance destroy <instance-id>`.

Notes on the rationale. Step 4 is the whole test. If any operator action
is needed between request and online, the requirement has failed even
though the instance works. Step 11 is checked by hand here and by an
automated test in `api/internal/enroll/`; the manual check exists because
log redaction failures usually appear at a layer no unit test observes.

If the bridge is unavailable during steps 5 to 9, that is a degradation
and not a failure. The instance is still created, still metered and still
expiring; enrollment queues and retries, and the instance is flagged as
un-enrolled. Blocking capacity on the trust plane would make capacity
unavailable for a reason unrelated to capacity.

**`enroll adopt` is not `instance reclaim`.** The two look similar and do
opposite things to the meter, so the distinction is worth stating once.
`compute-manager enroll adopt <address>` takes a machine the operator
already owns and pays for elsewhere, and enrolls it as a node. It writes
no instance, no intent and no reservation, so it is never metered and
stays free forever (`COMPUTEM-P1-001`). `compute-manager instance
reclaim <finding-id>` takes an instance this scenario created and lost
the record of, and rebuilds that record: it links the surviving intent,
opens a usage window from the provider's reported creation time, and
gives the instance an expiry from that point. Reclaim starts the meter.
Adopt never does.

### Respond To An Expiry Sweep Failure

**Owns:** `COMPUTEM-P0-004` (expiry enforced twice).
**Answers the alert:** "Instance past expiry and still running" in
[`OBSERVABILITY.md`](OBSERVABILITY.md#alerts--health), and the
`Expiry sweeper last-success age` signal that should fire before it.
**Status: not implemented, never exercised.**

This is not the drain drill above. The drill removes the sweeper on
purpose to prove the instance-side timer. This procedure is what to do
when the sweeper stopped without anyone asking it to, which is the case
where nobody knows how long instances have been running past expiry.

The failure is quiet by construction. A sweeper that raises no error and
does no work reports the same green health as a sweeper with nothing to
do, so the age of its last success is the only signal that separates the
two.

Perform these steps:

1. Run `compute-manager expiry list --json` and record every instance
   whose expiry is already in the past.
2. Read the sweeper's last-success age from the same output. Treat an
   age longer than one sweep interval as a stopped sweeper, not a slow
   one.
3. Do not restart the scenario yet. A restart resets the evidence and the
   next sweep will destroy the backlog before anyone has read it.
4. Run `make logs` and search for the sweep. Establish which of the three
   it is: the loop is not running, the loop is running and every pass
   fails, or the loop is running and the query returns nothing it should
   have returned.
5. Run `compute-manager expiry run` to sweep once on demand, and read
   the result against the list recorded in step 1.
6. Confirm each recorded instance reaches a destroyed state.
7. For every instance in step 1 that the sweep did not settle, follow
   `compute-manager instance destroy <instance-id>` individually. A
   sweeper that skipped an instance once will skip it again.
8. Run `compute-manager reconcile run` and confirm no `local-only`
   finding remains for the instances just destroyed.
9. Record the elapsed overrun per instance in
   [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Overrun is the
   cost of the outage and it is the only number that says whether this
   was expensive.

Notes on the rationale. Step 3 is the step people skip. The backlog is
the diagnostic: its size says how long the sweeper has been dead, and its
contents say whether the sweeper failed on a class of instance rather
than on all of them. Step 5 uses the on-demand verb rather than a
restart, so the loop's own state stays observable. If the instance-side
first-boot timer has already powered instances off, that is the backstop
working and not a reason to relax; a powered-off instance still bills on
most providers, so step 7 is what actually stops the meter.

### Respond To A Reservation Settle Failure

**Owns:** `COMPUTEM-P0-006` (credit reserved before boot).
**Answers the alert:** "Reservation heartbeat failing" in
[`OBSERVABILITY.md`](OBSERVABILITY.md#alerts--health).
**Status: not implemented, never exercised.**

An instance is running outside its reserved credit. Either a
re-reservation failed while the instance kept running, or a teardown
failed to settle measured usage against an open reservation. Both mean
this scenario is spending money it has not accounted for, and the second
also means the business suite is holding a reservation that will never
close on its own.

Perform these steps:

1. Run `compute-manager meter reservations --json` and separate open
   reservations from settled ones.
2. Cross-check against `compute-manager instance list --json`. Three
   divergences matter and they have different responses:
   - A running instance with no open reservation. It is uncovered and
     billing. Go to step 3.
   - An open reservation with no running instance. Credit is held
     against nothing. Go to step 5.
   - A running instance whose reservation covers less than its elapsed
     usage. Go to step 3.
3. Decide whether the instance should keep running. Uncovered capacity
   is a business decision, not an operational one, so an operator makes
   it and the runbook does not.
   - To keep it, add credit for the tenant in
     `landing-page-business-suite`, then run
     `compute-manager meter usage --instance <instance-id>` and confirm
     a reservation reopens on the next heartbeat.
   - To stop it, run `compute-manager instance destroy <instance-id>`.
     Destroy is the only stop; there is no pause that reduces the bill.
4. Confirm the tenant's ceiling in `compute-manager meter ceiling`
   before adding credit. A refusal that was correct is not an incident,
   and reopening a reservation past a ceiling converts a working control
   into an outage.
5. For a reservation held against nothing, settle it against the
   measured usage this scenario actually recorded, never against the
   reservation amount. Escalate to `landing-page-business-suite` with the
   reservation identifier and the instance's transition history.
6. Check whether the business suite was unreachable during the window.
   Provisioning fails closed on an unreachable business suite, so an
   unreachable suite explains a failed settle but never explains an
   uncovered running instance.
7. Record every uncovered instance-hour in
   [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

Notes on the rationale. Never settle against the reservation amount.
Usage is metered from the state transitions this scenario caused, and
that meter is the only record that survives a business-suite outage; a
settle that copies the reservation figure launders a guess into an
invoice. Step 3 refuses to destroy automatically for the same reason the
reconciler refuses: a running machine may be carrying work, and a
metering defect is not evidence that the machine is unwanted.

### Respond To A Provider Outage, Rate Limit Or Timeout

**Owns:** `COMPUTEM-P0-001` (provider-agnostic instance lifecycle) and
`COMPUTEM-P0-002` (intent recorded before the provider is called).
**Answers the signal:** `Provider adapter reachability` and the
`Lost-response rate` metric in
[`OBSERVABILITY.md`](OBSERVABILITY.md#signals).
**Status: not implemented, never exercised.**

The three symptoms are one procedure because the dangerous case is the
same in all three: a provider call whose outcome is unknown. An outage
that refuses cleanly costs nothing. A timeout that created a machine
costs money nobody can see, and it is the reason intent is written before
the provider is called.

Perform these steps:

1. Establish which one it is:
   - Refused with a provider error, and nothing was created. Safe.
   - Rate limited. Requests are being shed and may be retried.
   - Timed out, or the connection dropped mid-call. Outcome unknown.
2. Run `compute-manager provider list --json` and confirm which adapters
   report unreachable. An unreachable provider blocks new capacity and
   affects no existing instance.
3. Run `compute-manager intent list --state creating --json`. Every
   intent still in `creating` past its expected transition is a possible
   lost response.
4. Run `compute-manager reconcile run`.
5. Read `compute-manager reconcile findings --json` and match each
   `provider-only` finding to an intent from step 3 by its idempotency
   key. A match means the provider created the machine and this scenario
   never heard about it.
6. For each match, follow
   [Quarantine An Unaccounted Instance](#quarantine-an-unaccounted-instance)
   from step 4. That procedure's ownership branch is exactly this case,
   so reclaim is the expected outcome rather than the exception.
7. Do not retry a request whose outcome is unknown without its original
   idempotency key. A retry that carries a new key is how one lost
   response becomes two machines.
8. For a rate limit, stop issuing requests to that provider until the
   window resets. Note the limit in
   [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md); a rate limit hit
   during normal operation is a capacity-planning fact, not a transient.
9. Announce nothing about existing instances. They keep running, keep
   metering and keep expiring; a provider control-plane outage does not
   stop the instance-side expiry timer, which is the point of enforcing
   expiry twice.

Notes on the rationale. Step 7 is the whole procedure compressed into one
line. Idempotency is what makes a retry safe, and the key lives on the
intent precisely so a retry after an unknown outcome can carry the
original. Step 9 matters during an incident because the instinct is to
protect the fleet, and the fleet does not need protecting: the failure is
on the path that creates capacity, not on the path that runs it.

### Rotate A Provider Credential

**Owns:** no requirement. This procedure answers the credential
descriptor declared in `.vrooli/service.json` and the release gate in
[`DEPLOYMENT.md`](DEPLOYMENT.md), which will not pass until provider
credentials resolve through the credential authority.
**Status: not implemented, never exercised.**

Rotation is the case where this scenario's credential rule pays for
itself. The provider token is resolved by reference from the credential
authority at call time, so rotation replaces one stored value and touches
no code, no environment variable, no command argument and no row in this
scenario's database. If a rotation ever requires editing this scenario,
the rule has been broken somewhere and that is the defect to fix first.

Perform these steps:

1. Establish why. A scheduled rotation and a suspected compromise are
   the same mechanics in a different order, and the order is what
   matters. For a suspected compromise, revoke first and accept the
   outage. For a scheduled rotation, create the new credential first and
   revoke last, so there is no window with no working token.
2. Create the replacement token in the provider console with the same
   scope as the one it replaces. A rotation that quietly widens scope is
   a privilege change wearing a rotation's clothes.
3. Store it against the descriptor's logical identifier
   (`vrooli/compute-manager`, field `hetzner-api-token`) through the
   credential authority. Do not place it in `.env`, in
   `.vrooli/service.json`, in the CLI config file, or on any command
   line. `compute-manager configure` is not a credential store.
4. Run `compute-manager provider list --json` and confirm the adapter
   resolves and reports reachable.
5. Run `compute-manager provider describe <provider-id>` and confirm a
   real provider response, not a cached one.
6. Run `compute-manager reconcile run` as a read-only proof that the new
   token can enumerate provider inventory. Use the reconciler rather than
   creating a throwaway instance: it exercises the same credential, reads
   the whole account, and costs nothing.
7. Revoke the old token at the provider.
8. Run `compute-manager reconcile run` once more. A sweep that now fails
   means something was still holding the old token, and step 3 was not
   the only place it lived.
9. Search the logs for the old token value:
   `make logs | grep <last-six-characters-of-old-token>`. Expect no
   match. A match is a P0 defect in log redaction and the rotation does
   not close until it is fixed.
10. Record the rotation date. There is no automated expiry tracking for
    provider credentials in this scenario, so the record in
    [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) is the only
    thing that will remind anyone.

Notes on the rationale. Step 6 uses a read to prove a write credential
because the alternative, provisioning a machine to check a token, costs a
full billable hour on a provider that rounds partial hours up. Step 9 is
a check on this scenario rather than on the provider: the credential rule
says the value appears in no log line at any level, including on error
paths, and rotation is the one moment an operator holds both the old
value and a reason to search for it. Step 1's ordering is the only real
decision in the procedure, and getting it backwards during a suspected
compromise leaves an attacker holding a working token while the operator
performs a tidy rotation around them.

## Common Incidents

Every row is the intended response. None has been exercised, because the
behaviour that would produce the symptom does not exist yet.

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `compute-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Provisioning refuses with an out-of-credit result | Business suite reachability, tenant credit balance | Add credit, then retry the request | This is a correct refusal, not an incident. Escalate only if credit is present. |
| Provisioning refuses and the business suite is unreachable | Business suite `/health`, network reachability | Restore the business suite, then retry | Fail closed is deliberate. Do not add a bypass. A machine that boots unmetered is cost that grows hourly and cannot be recovered afterwards. |
| Instances are created but stay un-enrolled | `compute-manager enroll status`, bridge `/health` | Restore the bridge; the enrollment queue drains on its own | Expected while the bridge onboarding key endpoint is unpublished. |
| Reconciler reports a `provider-only` finding | `compute-manager reconcile findings` | Follow [Quarantine An Unaccounted Instance](#quarantine-an-unaccounted-instance) | Escalate if findings recur for the same provider identifier after resolution. |
| Reconciler reports a `local-only` finding | `compute-manager reconcile findings`, provider console | Close the usage window on the local record rather than continuing to meter | Escalate if metered usage continued past the provider's last-seen timestamp. |
| A provider call succeeded but the response was lost | `compute-manager intent list --state creating` | Run `compute-manager reconcile run` and let the sweep match the intent to the created instance | This is the failure mode intent-before-action exists for. Escalate only if the sweep cannot match. |
| Metered cost diverges from the provider bill | `compute-manager reconcile cost --day <date>` | Investigate the divergence; do not adjust the meter to match the bill | Provider billing lags by hours to more than a day. Only a sustained divergence is real. |
| A provider is unreachable, rate limiting, or timing out | `compute-manager provider list`, `compute-manager intent list --state creating` | Follow [Respond To A Provider Outage, Rate Limit Or Timeout](#respond-to-a-provider-outage-rate-limit-or-timeout) | Existing instances are unaffected. Escalate only if a lost response cannot be matched to a finding. |
| Instances are running past their expiry | `compute-manager expiry list`, sweeper last-success age | Follow [Respond To An Expiry Sweep Failure](#respond-to-an-expiry-sweep-failure) | Both enforcement paths have failed if the instance is also still powered on. |
| A running instance is not covered by a reservation | `compute-manager meter reservations`, `compute-manager meter ceiling` | Follow [Respond To A Reservation Settle Failure](#respond-to-a-reservation-settle-failure) | Uncovered capacity is a business decision. An operator decides whether to fund it or destroy it. |
| A provider credential is suspected compromised | `compute-manager provider list`, provider console audit log | Follow [Rotate A Provider Credential](#rotate-a-provider-credential), revoke-first ordering | The worst outcome available in this scenario. Treat as P0 and read [`../internal/SECURITY.md`](../internal/SECURITY.md). |

## Backup / Restore

Compute Manager holds the only durable record of what has been bought and
what it cost. Losing that database does not stop the provider from
charging, so a backup gap here is a financial exposure and not only a data
loss. Restoring a stale copy is worse than restoring nothing, because a
stale copy makes live instances look unaccounted and makes destroyed ones
look live.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database (intents, instances, receipts, findings) | deferred | deferred | Define before the first real provider credential is configured, not before deployment. |
| Provider API credentials | not applicable | not applicable | Held by the credential authority, never in this scenario's storage. Nothing to back up here. |
| Reservation and settlement records | not applicable | not applicable | Owned by `landing-page-business-suite`. This scenario holds references only. |

After any restore, run the reconciler before trusting the inventory. The
bidirectional sweep is what turns a possibly stale local record set back
into a picture of what actually exists.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Review reconciliation findings | daily, once the reconciler exists | `compute-manager reconcile findings` |
| Compare metered cost against the provider bill | daily, once cost reconciliation exists (`COMPUTEM-P1-003`) | `compute-manager reconcile cost --day <date>` |
| Rotate the provider API credential | on the provider's rotation policy, and immediately on suspicion | [Rotate A Provider Credential](#rotate-a-provider-credential) |
| Re-read the provider's reselling terms per service | on provider terms change | A general agreement can be overridden by a per-service annex. Check the annex, not only the general terms. |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

Escalate outside this scenario when the problem is not a capacity or cost
problem:

| Problem | Owner |
|---|---|
| Node identity, pairing, scopes, dispatch, onboarding | `vrooli-bridge` |
| Public hostname, DNS or ingress for an instance | `tunnel-manager` |
| A scenario failing to deploy onto an instance | `scenario-to-cloud`, `deployment-manager` |
| Wallet, entitlement, invoice or refund behaviour | `landing-page-business-suite` |
| An agent exceeding what it may spend | `treasury` |

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) - deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) - logs, metrics, and health signals
- [`../internal/SECURITY.md`](../internal/SECURITY.md) - credential handling and open gaps
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) - common fixes
- [`../reference/configuration.md`](../reference/configuration.md) - runtime configuration
- [`../../PRD.md`](../../PRD.md) - operational targets these procedures validate
