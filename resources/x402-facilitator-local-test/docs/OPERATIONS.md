# Local test operations

This resource uses the same pinned upstream artifact as the shared facilitator, but its independent resource name gives it separate managed-service state, configuration and port. Its static signer reference is `$FACILITATOR_PRIVATE_KEY`; the actual value is supplied only from Credential Authority to the managed process environment. Same-user `/proc/<pid>/environ` visibility applies.

The configured chain is loopback-only (`http://127.0.0.1:18545`) and only the v2 EIP-155 exact scheme is enabled. Start the disposable Base fork first, then use the resource lifecycle. Stop this resource before stopping the fork. The shared `x402-facilitator` remains unmodified.
