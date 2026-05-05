# Post-Connect Notes

The React-Vite template now uses Connect-RPC for the notes proto contract across API, CLI, and UI, with the notes attachment upload documented and implemented as the deliberate multipart REST exception. Shared substrate landed in `api-core` (`connectx`, `blobstore`), `cli-core` (Connect client + upload helpers), and `api-base` (Connect-Web transport), and the prompt-manager skills now steer future agents toward Connect-RPC for proto-owned wire boundaries.

Pointers:
- Iter-1 archive: `docs/plans/react-vite-template-fitness-iteration-1-plan.md`
- Active execution plan: `/home/matthalloran8/.claude/plans/thanks-for-the-postgres-wise-hearth.md`
- Recommended archive target: `docs/plans/react-vite-template-connect-rpc-adoption-plan.md`
- Superseded pre-Connect tactical proposal: `scenarios/prompt-manager/store/teams/meta-optimization/notebook/template-fitness/react-vite-template/2026-05-04/ITERATION_2_PROPOSAL_PRE_CONNECT.md`

Validation completed on a disposable generated scenario, `template-connect-rpc-smoke`:
- API: `go vet ./...`, `go build ./...`, `CGO_ENABLED=0 go build ./...`, `go test -race ./...`
- CLI: `go vet ./...`, `go build ./...`, `CGO_ENABLED=0 go build ./...`, `go test -race ./...`
- UI: `pnpm install --ignore-workspace`, `pnpm strings:check`, `pnpm type-check`, `pnpm lint`, `pnpm test:coverage`, `pnpm build`
- Runtime smoke: lifecycle `make setup`, lifecycle `make start`, CLI `notes list/create/get/attach`, invalid create error path (`invalid_argument`), help commands, and direct Connect-JSON `Notes/Create` curl
- Cleanup: disposable scenario, relocated proto schema, generated Go/TS/Python artifacts, temporary files, and installed disposable CLI removed; `packages/proto && make generate` rerun

Execution deviations/residual notes:
- Connect-ES v2 no longer uses `protoc-gen-connect-es`; TypeScript clients use service descriptors from `protoc-gen-es` v2 with `createClient`.
- `@vrooli/api-base` is consumed as a packed local `file:` dependency by generated scenario UIs, so `packages/api-base/package.json` now has `prepack: pnpm build` to avoid stale ignored `dist/` artifacts.
- The strict protojson grep from the original plan is too broad for the final template shape: protojson remains in deliberate REST exception helpers/tests and the existing health REST endpoint test path. The notes CRUD path is Connect-RPC; multipart attachment metadata remains proto JSON by design.
- Health is now documented as an operational REST exception. It stays REST so lifecycle systems, load balancers, and simple curl probes can read it without a generated Connect client.
- Focused shared-package coverage was strengthened after the e2e gate. `api-core/connectx` reaches 100%, `api-core/blobstore` reaches 89.3%, and the new `cli-core` Connect/upload helpers cover success, defaults, invalid inputs, Connect errors, API errors, and a bounded large upload. Remaining uncovered `blobstore` branches are filesystem error paths that require brittle OS-permission setup; keep them as residual risk rather than adding fragile tests.

Next harness step: re-baseline the 6-scenario template-fitness harness against this post-Connect template, then write a fresh post-Connect iteration-2 proposal from the new measured cost profile. Likely candidates to re-price: documentation density, test infrastructure replication, observability boilerplate, and the documented operational REST exception for health.
