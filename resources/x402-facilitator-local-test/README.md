# X402 Facilitator Local Test

This is a distinct managed-service resource identity for proving the pinned facilitator against a disposable Base fork. It has independent resource state/configuration and listens on loopback port 14021. Its only configured chain is Base chain ID 8453 at loopback RPC port 18545.

The signer is resolved from Credential Authority identity `vrooli/x402-facilitator-local-test`, field `facilitator-private-key`, and injected into the supervised child as `FACILITATOR_PRIVATE_KEY`. Use only generated test material. Never provision a funded key, and never point this resource at a public RPC endpoint.

The original `x402-facilitator` remains the shared empty-chain service on port 14020. This resource does not mutate or restart it. Start/stop this proving ground through `vrooli resource start|stop x402-facilitator-local-test` only after the local fork owns port 18545.
