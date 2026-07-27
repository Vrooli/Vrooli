# Scenario-to-Desktop deployment

Scenario-to-desktop is the Tier 2 desktop deployment ramp. It generates an
Electron desktop wrapper for a Vrooli scenario and records the generated
artifacts, build state, preflight results, and local validation evidence.

## Operating boundary

The scenario runs desktop artifacts directly on the current host when that host
matches the target platform. Local interactive validation is captured through a
loopback-bound VNC/websockify session and durable evidence captures.

Remote OS validation is a separate Vrooli Bridge consumer integration. Bridge
currently supplies typed node dispatch, runs, and artifact-transfer seams, but
does not yet define a desktop-session or evidence-capture transfer protocol;
therefore a bridge job ID is not represented as local desktop evidence.

## Verification

Use `vrooli scenario start scenario-to-desktop` to start the scenario, then run
`vrooli scenario test scenario-to-desktop` for the server-owned full suite.
See the scenario's `docs/guides/DEPLOYMENT.md` for package deployment details.
