# Phase 14 adoption friction

## Scope

Phase 14 adds real runtime use of `@vrooli/react-component-library/Button/2` to
the Agent Inbox, Architecture Cartographer, and Agent Manager UIs.

## Findings and dispositions

| Surface | Friction | Disposition | Evidence |
| --- | --- | --- | --- |
| All three UIs | Scenario Dependency Analyzer approved the local package, but its apply path emitted `pnpm add --ignore-scripts --workspace-root @vrooli/react-component-library` without the `file:` source and attempted the public npm registry. The package is intentionally private and returned HTTP 404. | Fixed locally with the established `file:../../../packages/react-component-library` manifest reference and the existing source-resolver pattern. Restored the equivalent local package links required by TypeScript resolution. | SDA dry-run: approved/local-file; SDA apply: registry 404; `make build` passes for all three UIs. | 
| Agent Inbox | The shared Button source imports library peer/runtime helpers that are outside the isolated scenario resolver. | Fixed with the Vite source resolver and consumer-local aliases for `clsx`, `tailwind-merge`, and `lucide-react`. | `make build` passes; 4,952 modules transformed. |
| Architecture Cartographer | Same isolated-consumer resolution boundary as Agent Inbox. | Fixed with the same governed source-resolver configuration. | `make build` passes; 1,904 modules transformed. |
| Agent Manager | TypeScript checks package imports before Vite can apply its resolver. | Fixed by restoring the local package link alongside the Vite source resolver. | `make build` passes, including `tsc --noEmit`; 3,160 modules transformed. |

No adoption was abandoned. The only remaining action is to repair the analyzer's
local-file apply command so future consumers do not need the workaround.
