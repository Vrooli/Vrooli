# Browser Automation Studio adoption

`consumer-declaration.json` is the public, source-controlled Channel Manager adoption declaration. It contains a stable profile key and a scenario workflow reference only. It must never contain a BAS runtime profile ID, cookies, tokens, credential material, or proxy credentials.

Channel Manager reconciles the declared key to a protected BAS runtime profile through the operator-only automation assignment. The identity-to-runtime-profile mapping is durable operational state, not committed configuration. Each mapping requires a D-009 acceptance reference and `operator-gated` automation mode.
