# CF1 agent-held session authority

Treasury derives one funded EOA per budget from its operator root wallet. A
live mandate binds an agent subject and run to that derived address. The agent
generates a separate, short-lived secp256k1 session key and proves possession
when it requests the mandate wallet. Treasury stores only the session public
key. The agent key does not contain the budget wallet's private scalar and
cannot bypass the Treasury settlement API.

## Request protocol

1. Provision the budget wallet and live run-bound mandate through the
   operator surface. The normal Agent Manager run must carry the
   `treasury:write` capability and an Agent Manager-issued operation-token pair
   whose canonical grant covers the requested budget and operation.
2. Agent Manager resolves or mints a fresh secp256k1 session key under the
   run-specific Credential Authority identity. The private value never enters
   `Run.CustomEnv`, SQLite or evidence; it is added after caller environment
   filtering to the runner process and resolved again for continuations. The
   existing terminal credential cleanup owner deletes it when the run ends.
3. The Treasury CLI derives the public key and calls
   `AgentSpend.GetBudgetWallet` with the baseline run token plus operation
   token, budget ID, mandate ID, and
   `X-Vrooli-Treasury-Session-Public-Key: 0x04…` (65-byte uncompressed public
   key). The response contains the derived budget address, mandate generation,
   and expiry. A different key cannot replace the bound key; operator rotation
   retires the generation and requires a new binding.
4. The Treasury CLI reads the authorization and wallet again immediately
   before settlement. It creates no signature while the authorization is
   pending, refused, released, expired, or missing an expiry. It also refuses
   to sign if the mandate session's returned expiry has passed. Once approved,
   it signs the exact `AgentSettlementIntentMessage` and sends the 65-byte
   Ethereum signature in `X-Vrooli-Treasury-Session-Proof` on
   `AgentSpend.ReportOutcome`.

Treasury verifies baseline and operation-token pairing through Agent Manager's
existing `/api/v1/identity/verify` pair route. The verified operation claim is
passed through authorization and settlement so the canonical operation/target
grant evaluator remains authoritative; a baseline-only request cannot silently
fall back when an operation token was supplied.

The signed message is the literal domain separator
`Vrooli Treasury mandate session settlement v1\n` followed by JSON for
`AgentSettlementIntent`, in Go struct field order. The digest applies the
Ethereum Signed Message prefix to those UTF-8 bytes. Fields bind the
authorization, mandate, budget, derived wallet ID/address, network, agent
subject/run, settlement ID, instrument ID, idempotency key, amount, currency,
counterparty, session generation, and session expiry. Clients should use the
exported message/digest helpers rather than rebuild field ordering themselves.

## Admission and replay

The agent signature is checked against the active public key, mandate, run,
authorization terms, budget policy, and freeze state before the settlement
owner is called. A durable row binds each authorization to one intent digest,
settlement, instrument, and idempotency key. Only the same intent can recover
after a lost response or process restart. The native-USDC signer independently
rechecks mandate, approval, expiry, allowlist, freeze, active session generation,
and the durable agent proof under the transactional freeze gate before creating
an EVM transaction signature. The settlement owner retains exactly-once
dispatch and reconciliation.

The agent-held signature proves authority over a specific settlement intent;
the EVM transaction itself is still signed by Treasury's protected derived
budget wallet. A denied or frozen intent cannot reach that signer. Operator
withdrawals continue to use the existing operator-authenticated canonical
mandate/authorization/settlement path.

## Evidence boundary

The local Base fork rehearsal generates the agent key in memory, signs both an
in-cap and human-approved over-cap intent, rejects an invalid signature without
changing the fork nonce, blocks the next request while frozen, and recovers the
same approved settlement after a normal Treasury restart. It stores public
keys, intent digests and signature digests in the isolated evidence folder; no
private agent key is written there.

The Agent Manager lifecycle, Treasury CLI signer, and paired-verification path
now implement the normal-run integration surface. Focused unit tests exercise
scope gating, protected injection, terminal secret cleanup, canonical JSON
signing, expired authorization/session no-sign behavior, and identity-pair
transport. The retained local-fork showcase still uses its owner test harness
and has not yet been repeated through an ordinary Agent Manager run and CLI
invocation. The session key is never suitable as an
EVM funding address or a replacement for an on-chain smart-account policy.
Mainnet/value-bearing operation remains gated by the operator runbook.
