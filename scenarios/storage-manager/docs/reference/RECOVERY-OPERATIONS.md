# Recovery operations

Use the server-owned recovery controller when the host needs free space.

1. Read the current pressure and writer evidence.
2. Start one recovery run.
3. Wait for the returned run id.
4. Read the terminal ledger row and action receipts.

```bash
storage-manager storage writers --top 10 --json
storage-manager cleanup run --trigger manual --json
storage-manager cleanup wait --run-id <run-id> --json
storage-manager cleanup history --limit 20 --json
```

The controller applies bounded batches in rung order. It re-checks free space
before each batch and stops at the configured target, a rung budget, the
operator line, or a safe error. It holds `recovery.lock` while it deletes.

Retention and recovery have different purposes. Retention enforces an owner's
declared age or byte budget on schedule. Recovery responds to pressure and
selects only providers admitted by the current authority rung. A standing
approval is required for conditional providers and is valid only for its
recorded host and subject.

Add a root by declaring its logical storage class, owner, regenerability proof,
containment, and retention budget in the repository contract or owner manifest.
Do not add a physical path to a scenario source file. Preview the resulting
provider before enabling it.

If a run reports `operator_line`, review the standing approval and provider
receipt before authorizing more reclamation. Never delete a protected,
durable, model, secret, or active-lease path to meet a percentage target.
