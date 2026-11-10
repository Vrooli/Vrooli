# CF4 local-fork rehearsal

**Status: partial CF4 evidence.** The local Treasury x402 rail, Offer Desk price
lookup seam, and ambiguous-response replay have a local Base-fork proof. This
does not close CF4: the configured upstream facilitator is still fail-closed,
the Offer Desk catalog response in this rehearsal is a Connect test fixture,
and the test-priced HTTP resource is a local rehearsal endpoint.

## Implemented path

- Treasury's x402 price declaration reads a positive declared USD sells_at
  edge from Offer Desk's generated Connect CatalogService/ListEdges client.
  The operator-authorized POST /api/v1/x402/prices/offer-desk route uses that
  quote to set the price amount; callers cannot override the amount.
- The public admission route accepts GET, which matches Treasury's outbound
  rail's x402 request method; POST remains supported for existing callers.
- The outbound x402 rail is exercised against the real Base USDC EIP-3009
  contract on Treasury's disposable Anvil fork. A test-memory payer key signs
  the exact one-cent amount passed by the rail caller. The test does not build
  a real mandate or authorization through Treasury's settlement service. Anvil mints only on the local
  fork, using its impersonated test minter.
- A test-only cryptographic facilitator recovers and checks the EIP-712 payer,
  submits transferWithAuthorization on the local fork, then drops its first
  receipt response after the transaction commits.
- The rail callback stores the exact signed payload in an encrypted,
  test-scoped Credential Authority file and verifies the digest/reference
  boundary in the test. A new rail adapter resolves the payload back from Credential
  Authority and replays those exact bytes. The test checks the matching Base
  USDC authorization nonce is consumed and the original transaction hash is
  returned.
- No private key or signed payload is printed or committed. The test-scoped
  Credential Authority file and local fork are removed by test cleanup.

The test facilitator is not the digest-pinned Second State service. It performs
real EIP-712 recovery and the token contract performs the actual local-chain
transfer, but the upstream facilitator's /verify and /settle endpoints were
not called in this rehearsal.

## Retained run

Focused fork command:

    cd scenarios/treasury/api
    TREASURY_BASE_FORK_TEST=1 go test ./internal/rail/x402 -run '^TestCF4LocalBaseForkReconcilesExactPayment$' -count=1 -v

Result: PASS. The pinned upstream snapshot was Base block 55331671 with hash
0x2a0665b8deba215f5ea7e31be6658fd11859f66eee1249020f22461c64233b01.

    CF4 reconciliation: outcome=settled basis=x402_facilitator_replay_confirmation
    CF4 local Base fork: block=0x34c4b57 block_hash=0x2a0665b8deba215f5ea7e31be6658fd11859f66eee1249020f22461c64233b01 per_call_cap_minor=1 amount_atoms=10000 payload_digest=sha256:c823355093b159375852b3723579a05b646cc305f935e4adc19885b2319ee785 nonce=0x8ae660a12a0599965ae70f42c7347d712b65037dcf8acd3f19f19f2a461dbc20 nonce_consumed=true tx=0xa28482e1aff5bff6a4e07e1588ffe32c9d63440b6f9612ec51a1acb522e19040 ambiguous_receipt_reconciled=true
    --- PASS: TestCF4LocalBaseForkReconcilesExactPayment

The retained digest is over the exact base64-encoded signed payment bytes. The
exact payload remains in Credential Authority during the test, not in this
evidence file. The reconciliation exercises a reconstructed rail adapter; it
does not restart the Treasury process or Credential Authority daemon.

Offer Desk and gate source checks:

    cd scenarios/treasury/api
    go test ./internal/rail/x402 ./handlers/x402gate -count=1

Result: both packages PASS. TestCF4LocalBaseForkReconcilesExactPayment obtains
its one-cent USD price through the Offer Desk Connect client before generating
the local endpoint's payment challenge. The handler test verifies
/api/v1/x402/prices/offer-desk requires operator authorization and ignores
caller-supplied price values in favor of the Offer Desk response.

Treasury Test Genie run 20261109-205850-67a4f88e passed its unit phase
(24 seconds, 43 observations). The scorecard still reports the existing
MISSING_INJECTABLE_SEAM warning and 39 low-coverage warnings; this scoped run
does not certify the full Treasury scenario.

## Upstream facilitator and platform status

The live resource was already running and was left unchanged. Read-only
inspection returned healthy on artifact version 1.3.0, with GET /supported
returning {"kinds":[],"extensions":[],"signers":{}}.
resources/x402-facilitator/resource.json continues to declare Linux supported
and macOS/Windows unsupported because upstream publishes no signed native
artifacts for those platforms.

The pinned upstream configuration requires a signer reference in a chain
signers list and expands it from the facilitator process environment (see the
pinned facilitator README:
https://github.com/second-state/x402-facilitator/blob/e75adda8a0e3cf23db446883ba91b88bbab2fe28/facilitator/README.md).
The resource has no Credential Authority-to-signer injection mechanism. Since
this work may not put signer material in argv, environment, logs, or the
repository, the upstream service remains on empty chains and schemes.
This is a concrete integration gate, not a claim that the upstream service has
settled on Base.

## Still open

- Wire the Offer Desk pricing route to a maintained Vrooli priced endpoint and
  rehearse inbound settlement against the upstream facilitator when a safe
  signer lifecycle exists.
- Prove Treasury's production SQLite settlement row and Credential Authority
  recovery reference across process restarts, then reconcile by on-chain
  transaction/nonce without creating a new signature.
- Add the upstream facilitator's signer through a CA-backed path that keeps
  material out of process environment, argv, logs, status output, and files,
  or replace the upstream with an implementation that supports that contract.
- Complete the full CF4 local-fork showcase and optional Base Sepolia route
  where a no-account faucet is available.
