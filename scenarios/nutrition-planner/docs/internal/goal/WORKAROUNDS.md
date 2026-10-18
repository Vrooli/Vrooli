# Nooch infrastructure workarounds

## E30 interactive launch is unavailable for retained identity
SUP-IDENTITY-RECOVERED-20261014 resolved the identity hold: current native readback gives retained parent run `16d59037-b9a9-4442-8ae7-0c28921bd54b`, task `27847f20-3191-47d9-b74e-76df6ef918f8`, empty subject and zero scopes; AM serves build `sha256:0172797354f20ccb7d3fe67f30f1fa1cd05fc5450fe4531dee1a0846e3c934a6`; final broad unit `20261014-024937-9072cd54` passed. The original Planner consumer qualified identity/admission using a sandboxed codec-pipe inspection profile, not the E30 interactive epoch-worker profile. E30 create run `02e08dc1-0ef8-4a4c-9fb2-9b309e2220ce` on the unchanged task/profile/parent failed before worker start: `web-console stable-create identity unsupported; no Create submitted`. No Create was submitted and task `51b75231-d745-4279-9aed-435b43c439a4` remains queued. Keep dispatch held pending supported registered interactive stable-create for this retained identity. Do not alter credentials, profiles, grants, headers, execution mode or launch route. E30 scope, anchor, model, workload and gates remain unchanged.

## E3 exact-build capture route remains unavailable
The earlier CaptureService action was denied. Resume only after the existing BAS/CaptureService owner returns a qualified route, exact current build and authorized fixture/state. Do not substitute or retry another route.

## E24 review remains external
Run `cca3f82e-5564-4ca3-bd8f-a369993322fb` is `NEEDS_REVIEW` because its Web Console session was lost; continue/recovery was refused without operator lifecycle-recovery context. Preserve its result and proposal; do not reset its baseline or create a replacement leader.

## Archive
Prior E25/E30 runner incidents, run-control evidence and detailed receipts remain in `archive/WORKAROUNDS-before-continuation-20261014T0235Z.md` and `archive/WORKAROUNDS-before-resume-compaction-20261007T1157Z.md`.
