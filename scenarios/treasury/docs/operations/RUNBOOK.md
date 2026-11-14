# Runbook — Treasury

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

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

## Scoped Card Sandbox Activation

The `lithic-sandbox-card` adapter is intentionally restricted in code to
`https://sandbox.lithic.com` (and loopback test servers). It cannot point at
Lithic's production origin. The following steps therefore enable test money
only; production card issuance is not shipped.

Steps 1–3 require an account and credential that only the operator can create:

1. Create or sign in to a Lithic account, select the **Sandbox** environment,
   and obtain an Issuing API key with permission to create virtual cards and
   V2 Authorization Rules. If the program requires account-level card
   association, also copy its sandbox `account_token`.
2. Confirm the credential store is healthy with `vrooli credentials doctor`
   and initialize/unlock it if the command says that is required.
3. From a trusted secret source, pipe one strict JSON object to the credential
   authority. Never put the JSON literal in shell history:

   ```bash
   your-secret-source-command | vrooli credentials provision \
     --identity vrooli/treasury/lithic --field value
   vrooli credentials status \
     --identity vrooli/treasury/lithic --field value --format json
   ```

   The value has shape `{"api_key":"…","account_token":"…"}`;
   `account_token` is optional. The status command must report configured and
   never prints the value.

Before issuing any card, complete the SQLite backup prerequisite in the table
below. Then create a live mandate whose counterparty is the provider's merchant
acceptor ID (15 characters or fewer), and register its instrument:

```bash
treasury instrument register --instrument \
  '{"id":"card-instrument-1","mandateId":"<mandate-id>","rail":"lithic-sandbox-card","credentialReference":"vrooli/treasury/lithic","counterparty":"<merchant-acceptor-id>"}'
```

Registration resolves the provider API key only for the issuance call. The
provider receives the mandate cap as a lifetime card spend limit, the mandate
counterparty as an active card-level deny rule for every other merchant ID,
and the mandate expiry month/year as the network card expiry. Treasury keeps
enforcing the mandate's exact timestamp; card networks represent expiry only
to month precision. The returned PAN/CVV envelope is immediately provisioned
under a derived `vrooli/treasury/instruments/...` identity. Treasury persists
only that logical reference, and the RPC response omits it.

To run the explicit live-sandbox proof, opt in for one run and then block once
using the run id returned by Test Genie:

```bash
TREASURY_LITHIC_SANDBOX_INTEGRATION=1 vrooli scenario test treasury
test-genie runs wait --json treasury <run-id>
```

Read `TestLithicSandboxProviderEnforcesMandateScope` in the unit phase. It
issues a fresh sandbox card, reads the provider's card and active merchant
rule back, then sends an over-cap authorization and an authorization for a
different merchant ID. Both must be definite provider declines. If the test is
skipped, `TRS-P1-003` is not live-verified, regardless of local unit results.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `treasury status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| **All automated spend is refused** | Is `agent-manager` running? `vrooli scenario status agent-manager` | Start `agent-manager`. Automated spend resumes with no other action; nothing was recorded at a degraded grade while it was down. | This is the fail-closed decision working as designed, not a defect. See `../internal/DECISIONS.md`. |
| **A settlement is stuck in `unknown`** | The charge record and the rail's own status for that reference | **Query the rail. Never retry.** Retrying an unknown is how a double charge happens. Resolve the state from the rail's answer. | If the rail cannot answer, leave it `unknown` and escalate. An unresolved `unknown` is safer than a guessed outcome. |
| **Money moved but the ledger does not show it** | Emission log; is `money-ledger` reachable? | Emission retries on its own. Local evidence is authoritative until it succeeds. | If the backlog persists past one retry cycle, financial position is wrong downstream — escalate. |
| **An agent reports it cannot buy something** | The refused attempt in Activity; the named constraint | Not an incident. Read which constraint refused, then decide whether to widen it. | Only escalate if the refusal names no constraint, which would be a defect. |
| **A charge appeared that nobody expected** | The attempt's evidence record: mandate, approver, request | Freeze the relevant budget first, investigate second. The freeze binds before the next authorization. | Treat as a security incident until the evidence shows an ordinary authorization. |
| **Approval requests are not arriving on a device** | `notification-hub` reachability; relay attempt records | None required — the console queue is unaffected and approval still works. | Relay is an enhancement, never a dependency. |
| **Scoped-card issuance is unavailable** | Credential status for `vrooli/treasury/lithic:value`; Lithic sandbox account permissions; API logs | Restore/unlock Credential Authority or correct the operator-created sandbox key/permissions. Do not pass a key in an RPC or environment variable. | Leave the rail disabled and use the manual rail. A provider card may have been created before a later rule/provisioning failure; retry the same instrument id so provider idempotency converges instead of issuing another card. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

**Backup matters more here than in an ordinary scenario.** The evidence
journal is the artefact that proves what was authorized; losing it does not
lose money but does lose the ability to account for money. Treat it as the
primary restore target rather than an afterthought.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database (evidence, mandates, charges) | Register with `data-backup-manager` before any automated rail is enabled. | Restore the file and restart through the lifecycle. | **Required before the first automated rail.** Not yet configured. |
| Idempotency keys | Included in the database backup. | Included in the restore. | Restoring a database *without* them would make in-flight client retries into potential double charges. Never restore a partial database. |
| Instrument credentials | Not applicable — held in `secrets-manager`, backed up there. | Re-resolved by reference after restore. | Deliberate; nothing to back up here. |
| Blob files | not-applicable | not-applicable | This scenario stores no blobs once the example domain is removed. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Review pending approvals | daily while agents are active | Console landing page; an aged pending request means a stalled agent. |
| Check for `unknown` settlements | daily while an automated rail is live | Activity page. Any non-zero count needs a human. |
| Verify the freeze works | before trusting a new rail | Freeze, attempt an authorization, confirm refusal, thaw. A kill switch never exercised is a kill switch never verified. |
| Reconcile against rail statements | monthly once an automated rail is live | Manual until `TRS-P2-001` lands. |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration

## Receive-only research wallet qualification

The source exposes `wallet create`, `wallet get`, `wallet prove`, `wallet qualify-recovery` and `wallet recover` through TreasuryAdmin. Use an established authenticated human operator ingress; agent-attributed and mixed-realm calls must retain their refusal. Production commissioning through the current agent session has not been qualified.

For an operator commissioning session, start with Base Sepolia (`eip155:84532`) and a book whose beneficiary is the verified operator. Keep one stable wallet ID for retries. Creation persists its opaque reference before minting, so retrying a partial response must recover the same binding. A mainnet wallet is a separate explicit wallet ID/network (`eip155:8453`), never an automatic promotion of the testnet record.

After creation, read the binding again, produce a fixed non-payment proof with a fresh 16–128 character alphanumeric/underscore/hyphen nonce, verify its address/public key/message/network, then qualify encrypted recovery. Reopening the owner must return the same public address/reference/recovery hash. A wrong-network read must refuse. These operations do not create any spending budget or authorize an outgoing transfer.

`same_host_encrypted_verified` means the encrypted bundle was persisted and its scoped contents restored in transient owner memory and matched to the address. It does not mean a replacement host can recover it. Once an operator has commissioned the wallet, the control-plane Credential Authority can create an encrypted recovery bundle that includes configured Treasury wallet credentials:

```bash
vrooli credentials recovery export --all \
  --output <private-path-on-approved-off-host-volume>/vrooli-recovery.bundle
vrooli credentials recovery verify \
  --input <private-path-on-approved-off-host-volume>/vrooli-recovery.bundle
```

`--all` covers all configured credentials in the current inventory, not only Treasury. Treat the bundle accordingly; enter a strong operator-held passphrase at the secure prompt and store that passphrase separately from the bundle. Never put it in this runbook, a command argument, or the bundle's directory. The `verify` command checks the encrypted bundle; it does not prove a replacement-host restore, wallet-address match, or owner-only signing. Test restore only in the approved replacement-host recovery exercise, then confirm Treasury returns the same wallet binding and a fresh operator control proof verifies. Separately use Data Backup Manager to back up and restore Treasury metadata and its encrypted wallet recovery file to a clean compatible target. Do not mark recovery READY until both credential and data restore evidence pass.

`wallet recover` restores only a missing, previously witnessed scalar after staged encrypted contents match the recorded address; it refuses an existing mismatched key or lost passphrase. Qualification and missing-key recovery reject a changed qualified bundle by its recorded checksum before opening its envelope. RPC/CLI generation and focused consumer checks passed; production commissioning and host recovery remain unverified. Preserve older wallet records and custody references when creating a replacement wallet ID; never replace a key behind an existing address binding.

For future Base funding, Circle-native mainnet USDC is `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913` with six decimals. Sepolia's test token is a distinct contract and has no research purchasing value. A balance read is a snapshot; a transfer requires an exact canonical occurrence and owner/valuation evidence before Money Ledger acceptance or Treasury allocation. Direct outgoing transactions need native ETH gas; sponsored ERC3009 support requires its own qualified provider path. Current wallet qualification supplies no receiving address approved for funding.

### Inspect an inbound Base USDC transfer

After the operator has independently qualified wallet recovery and selected
the exact transaction in Coinbase's Base app, an authenticated Treasury
operator can run:

```bash
treasury wallet receipt-check --wallet-id <wallet-id> --transaction-hash 0x<64-hex>
```

This read checks the operator-bound Treasury wallet, Base mainnet chain ID,
Circle's native USDC contract, successful receipt, exact ERC-20 `Transfer`
logs to that wallet, canonical block hash, and the RPC's `finalized` head. It
returns exact six-decimal atomic units and provider/finality evidence. By
default it uses the public Base RPC; `TREASURY_BASE_RPC_URL` can select an
operator-managed HTTPS endpoint. HTTP is accepted only for loopback. The RPC
client allows only `eth_chainId`, `eth_getTransactionReceipt`, and
`eth_getBlockByNumber`, refuses redirects, and does not log the endpoint.

Treasury appends each check result and successful transfer observation to its
own database. Read the retained history for the same operator, wallet, and
transaction with:

```bash
treasury wallet receipt-checks --wallet-id <wallet-id> --transaction-hash 0x<64-hex>
```

`included` means the provider has not yet placed the block at or below its
finalized head; wait and check again. Each check is retained as a new
observation, so later evidence does not overwrite an earlier inclusion. Even
`finalized` is one RPC provider's report, not independent consensus proof. A
successful receipt check does not prove off-host wallet recovery, value the
token in USD, create an accounting posting, add budget credit, authorize
spending, or make the funding handoff READY. The exact atomic token amount
remains separate from any USD valuation or Money Ledger entry.
