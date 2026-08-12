# session-core

`session-core` is the shared, transport-neutral interactive-session contract.

It owns the typed PTY lifecycle (`Read`, `WriteInput`, resize, close and kill)
and the interchangeable local, authenticated node-agent and SSH backend seams.
The bridge WebSocket and cloud terminal transports remain responsible for
authentication and wire encoding; they must not add a raw shell-command path.

The agent and SSH adapters intentionally accept narrow interfaces. That keeps
the security policy and byte protocol stable while allowing the desktop agent,
bridge channel and scenario-to-cloud to use their native transport libraries.
