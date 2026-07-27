# Scenario-to-Desktop browser workflows

The six cases exercise the user-facing generator, preflight, build, smoke-test,
signing, and live-desktop prerequisite surfaces against the running scenario.
They use stable selectors in `ui/src/constants/selectors.ts` rather than visual
classes or copy.

The current catalog is intentionally observer-only: generating wrappers,
building installers, saving signing configuration, and starting a VNC session
all mutate file-backed scenario state. Those actions must not run until the
scenario opts into a Test Genie routed file lease and has a deterministic
desktop-artifact fixture. The live-desktop case therefore records the honest
pre-artifact state instead of falsely claiming a VNC canvas was reached.

Once that fixture exists, add mutating cases labelled `requires_confirmation`
and `routed_isolation`, then assert the generated artifact, build result,
smoke report, and `@selector/liveDesktop.canvas` respectively.
