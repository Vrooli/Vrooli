# Scenario-to-Desktop browser workflows

The six cases exercise the user-facing generator, preflight, build, smoke-test,
signing, and live-desktop prerequisite surfaces against the running scenario.
They use stable selectors in `ui/src/constants/selectors.ts` rather than visual
classes or copy.

The catalog includes one deliberately scoped mutating case:
`04-evidence/leased-desktop-evidence.json`. Test Genie installs a routed file
lease, BAS supplies `X-Vrooli-Test-Mode`, and the console fixture refuses any
request without that active lease. It creates a deterministic fixture AppImage
and smoke report under the leased data root, then asserts the returned routed
write count. This proves the console mutation path, its lease, and its BAS
artifacts without writing normal scenario state.

The fixture is not a substitute for a real release. Real installer build,
launch, use, update, and quit evidence remains under the desktop-readiness
artifact directory. The live-desktop case continues to record its honest
pre-artifact state until a separately isolated VNC success fixture exists.
