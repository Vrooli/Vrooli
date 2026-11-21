# ai-money-fixtures

This standard-library-only package holds shared contract types and deterministic
fakes for the agent-connect AI money integration. It is test-only: a passing
contract test here proves that test doubles agree with the F1 shape, not that
LPBS, AI Gateway, Stripe, a provider, a harness, or desktop runtime behaves the
same way. Every fake must be run through the same contract suite as its real
counterpart before a product claim is accepted.

## F1 contracts

- Public errors use the stable fields `error`, `error_type`, and `retryable`;
  credits refusal is HTTP 402 with `credits_required`, while route refusal is
  `no_eligible_route`.
- A feature request carries its declared instruction, exactly one user turn,
  and schema. It cannot choose its own use class.
- The private LPBS-to-Gateway contract uses distinct plan, execute, and receipt
  types. It is not the public `GatewayRequest`. The funded marker is set by the
  trusted front-door fake after the reserve boundary; it is not accepted from
  the public body.
- A usage receipt keeps supplier cost and customer charge separate, and
  attributes the operation to funding `lot_ids` and `source_kinds`.

`testdata/usage-receipt.golden.json` is a serialization fixture. The limit
corpus is explicitly synthetic and unverified until replaced/paired with the
operator-authorized real captures; never report it as evidence of real provider
messages.

## Fakes

`FakeFrontDoor`, `FakeGatewayPrivate`, and `FakeRouter` provide deterministic
HTTP/Go seams. `FakeStripe` carries payment, asynchronous payment, refund,
dispute, and early-fraud-warning events but does not emulate Stripe signatures
or delivery retries. `StaticKeyIssuer` and `NewSignedComponentFixture` use
fixture-only deterministic keys. `FakeLifecycleBackend` models two explicit
backends and is not a production lifecycle implementation.

No fake authorizes a provider, approves rule records, sets trust/prices, or
initiates a real payment. Real component and integration proof is still required.
