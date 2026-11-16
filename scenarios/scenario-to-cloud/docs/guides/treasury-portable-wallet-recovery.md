# Treasury portable wallet recovery through Scenario-to-Cloud

Scenario-to-Cloud consumes Treasury's encrypted, key-only portable recovery bundle through the owning Data Backup Manager (DBM) and Treasury command surfaces. DBM treats the bundle as opaque ciphertext. Treasury alone decrypts it with the operator-held recovery secret. The command never puts that secret in argv, a file, or its output.

The DBM source is the per-wallet directory reported by Treasury's `export-portable-recovery` response (`relative_path`). It contains `wallet.bundle` (encrypted key-only ciphertext) and `witness.json` (public metadata authenticated with the same operator-held secret). Register that directory as the filesystem target so DBM snapshots both files together. Rotation stages `revocation.json`, signed by the old wallet key, and removes the live bundle. The operator-context Scenario-to-Cloud `wallet-recovery rotate` command publishes that same staged tombstone through a manual DBM plan, checks that the run produced the newest snapshot, verifies and scratch-restores it, and passes Treasury a receipt bound to the exact tombstone digest and snapshot. Treasury retains the old scalar until that receipt validates, then commits revocation and deletes it. Without a healthy writable destination and verified readback, rotation remains pending. Use the same stable successor ID and tombstone after a retry. The witness is not trusted by itself: Treasury checks its MAC, validates the deterministic custody reference and beneficiary book, decrypts the bundle transiently, and matches the recovered key to the witnessed address and public key before creating wallet metadata. A clean node can create the beneficiary book from the authenticated public witness, but any existing conflicting book or wallet binding is refused. Existing keys are never replaced. Restore into a new, private staging directory on the recovery node. Do not use a second directory on the source host as the off-host repository.

Rotation requires an existing qualified S3/SFTP destination and a manual, disabled plan containing exactly the Treasury wallet recovery target and that destination. It refuses local destinations and does not create credentials or choose a destination. Before any live use, verify that the destination and its existing operator credentials are available to DBM.

```bash
scenario-to-cloud wallet-recovery rotate \
  --wallet-id <wallet-id> \
  --network <network> \
  --successor-id <stable-successor-id> \
  --destination-id <existing-off-host-destination-id> \
  --plan-id <manual-disabled-plan-id> \
  --staging-dir "$HOME/wallet-rotation-verify" \
  --receipt-file "$HOME/wallet-recovery-evidence/rotation.json"
```

Use a new private empty staging directory for the scratch restore, separate from the source target. The receipt contains metadata and digests only. A failed or interrupted publish never authorizes scalar deletion; rerun with the same successor ID after confirming the DBM plan and destination, and provide a fresh empty staging directory and a new receipt path for each attempt.

## Fetch and restore

On the recovery node, create a private empty directory and provide the exact destination metadata. Scenario-to-Cloud enumerates snapshots for the stable Treasury target identity and selects the newest one:

```bash
mkdir -m 700 -p "$HOME/wallet-recovery-stage"
mkdir -m 700 -p "$HOME/wallet-recovery-evidence"
read -r -s -p 'Operator recovery secret: ' WALLET_RECOVERY_SECRET
printf '\n'
printf '%s' "$WALLET_RECOVERY_SECRET" | scenario-to-cloud wallet-recovery restore \
  --wallet-id <wallet-id> \
  --network <network> \
  --bridge-node-id <durable-bridge-node-id> \
  --bridge-machine-id <durable-bridge-machine-id> \
  --bridge-scope scenario-to-cloud:read \
  --destination-id <destination-id> \
  --destination-name <destination-name> \
  --backend s3 \
  --location '<existing-s3-repository-location>' \
  --created-at <original-destination-created-at-rfc3339> \
  --staging-dir "$HOME/wallet-recovery-stage" \
  --evidence-file "$HOME/wallet-recovery-evidence/restore.json" \
  --secret-stdin
unset WALLET_RECOVERY_SECRET
```

For SFTP, pass `--backend sftp` and its existing non-secret repository location. The SFTP/S3 credentials must already be available to DBM on the recovery node. The command imports the exact destination metadata, lists snapshots for the stable `dbm://treasury/treasury-wallet-recovery-<wallet-id>` identity, selects the newest timestamp, registers the private staging directory under that identity, then verifies and restores only the selected snapshot. If it contains `revocation.json`, recovery stops and never falls back to an older active wallet snapshot. Otherwise Treasury independently validates the bundle and witness before restoring. DBM and Treasury each enforce their own authorization and readiness rules.

The node ID and Machine ID are durable Bridge registry identities. Scenario-to-Cloud checks that `nodes get --json` returns that exact node ID with the exact registry-approved recovery scope, that `machines get --json` lists the node as the current lineage entry for that exact Machine, then requires `nodes doctor --json` to pass before DBM import or restore. A stale heartbeat, disconnected channel, mismatched node, or missing scope fails before Treasury sees the bundle or secret. Both identities and the scope are stored in the private evidence and must be supplied again after restart; the command rechecks live readiness before requesting Treasury's proof. Bridge authentication must already be configured locally (`vrooli-bridge configure token` or the supported token environment); no credential is entered during recovery.

The shell variable above exists only to pipe the operator's secret; avoid shell history or tracing (`set -x`). Use an interactive password manager or another operator-controlled stdin source when available. The agent must not request, read, retain or report a real recovery secret.

## Prove control after restart

Restart Treasury on the recovery node through the normal Vrooli lifecycle command. After it is healthy, compare against the address witnessed before source loss:

```bash
scenario-to-cloud wallet-recovery prove-after-restart \
  --wallet-id <wallet-id> \
  --network <network> \
  --bridge-node-id <same-durable-bridge-node-id> \
  --bridge-machine-id <same-durable-bridge-machine-id> \
  --bridge-scope scenario-to-cloud:read \
  --expected-address <witnessed-address> \
  --nonce <fresh-unique-nonce> \
  --evidence-file "$HOME/wallet-recovery-evidence/restore.json"
```

The command succeeds only when Treasury returns a verified signature for the expected address and fresh nonce. A DBM snapshot verification without this Treasury proof is not a successful wallet recovery. This read-control signature is not a payment or spending authorization.

## Evidence and limits

The private evidence file joins the newest DBM snapshot ID, destination and target identity, exact bundle and witness digests, Bridge node identity and approved scope, Treasury's confirmed restored wallet address, and the verified post-restart proof. `prove-after-restart` rehashes both unchanged artifacts, rechecks the newest DBM snapshot ID and rechecks the Bridge node before contacting Treasury; it refuses tampering, snapshot supersession, scope or node changes, stale readiness, and a repeated proof update. Keep this file with private operator evidence; it contains no recovery secret, repository credential, or bundle bytes. The bundle itself remains encrypted and private; the witness contains public metadata and a secret-keyed authentication tag. This consumer proves that Scenario-to-Cloud can fetch the Treasury recovery directory from an independently reachable DBM repository and submit both artifacts to Treasury; it does not prove a funded transfer or provider availability outside the exercised repository.

Both restore and post-restart proof require DBM's bounded read lease for the exact newest snapshot and generation. DBM excludes supported snapshot mutations and tombstone attestations until Treasury returns, renews the lease before the owner operation, then releases it; interrupted leases expire after at most 15 minutes. Lease routes verify a Scenario Authenticator bearer for the exclusive `data-backup-manager` audience and accept only a verified human operator. An explicit local `data-backup-manager wallet-recovery read-lease-*` command exchanges the existing local operator session for that audience in memory only after the local broker confirms owner eligibility; it neither provisions a service principal nor uses a configured bearer from a process carrying agent identity. Run the explicit lease command from the operator's local shell outside Agent Manager. Ensure the authenticator's `data-backup-manager` audience configuration and deployed DBM API are current before using this flow. The local SQLite lease coordinates only this DBM instance and its supported mutation paths; direct provider writes or another DBM instance are outside this contract.
