# session-core

`session-core` is the shared, transport-neutral interactive-session contract.

It owns the typed PTY lifecycle (`Read`, `WriteInput`, resize, close and kill)
and the interchangeable local and authenticated node-agent backend seams.

SSH is intentionally not a supported interactive-session transport. SSH
onboarding remains available in the Bridge control plane, but no session
backend is advertised until a credentialed, constructed implementation exists.
The bridge WebSocket and cloud terminal transports remain responsible for
authentication and wire encoding; they must not add a raw shell-command path.

The agent and SSH adapters intentionally accept narrow interfaces. That keeps
the security policy and byte protocol stable while allowing the desktop agent,
bridge channel and scenario-to-cloud to use their native transport libraries.
