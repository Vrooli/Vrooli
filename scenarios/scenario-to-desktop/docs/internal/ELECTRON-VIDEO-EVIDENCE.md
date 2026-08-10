# Electron video evidence handoff

Captured 2026-08-09 on the native Linux/Xvfb target after the live-desktop
recording path was hardened to validate the finished MP4 before persistence.

## Canonical producer captures

These are fresh `scenario-to-desktop pipeline run ... --stages
generate,build,smoketest` deliverables. Each is a generated Linux Electron
AppImage, not a browser-only or fixture-only recording.

### Scenario to Desktop

- Pipeline: `4d992263-9aee-63b1-42a6-56bb4e8d97ab`
- Artifact digest: `sha256:9ab0b7764c2b681fbac4ef19843a2f7eaf6d39d90605dd6e8d164e97a540b975`
- Smoke run: `smoke-scenario-to-desktop-1786312063706` (protocol passed; desktop journey passed)
- Capture: `28e189cd-4d3b-4191-93cd-e0bee23fc1b8`
- Capture URL: `/api/v1/captures/scenario-to-desktop/28e189cd-4d3b-4191-93cd-e0bee23fc1b8/file`
- Video: H.264 MP4, 1920x1080, 9.666 seconds, 748,399 bytes
- Video checksum: `sha256:72658771b3a09bc0c72a8a3c8829258ea83ab47bf8fecd4ed259abb5afa67ce2`
- Manifest: `recording-1786312074060.mp4.manifest.json`, state `passed`, all five required gates passed

### Secrets Manager

- Pipeline: `dbcbe219-cf34-3f6a-1506-d08c70a0d3f0`
- Artifact digest: `sha256:83eb0022a59f76a0dc92eb63271b36da41e8e6d9dabe4a62e30b134c8dd0c781`
- Smoke run: `smoke-secrets-manager-1786312444540` (protocol passed; desktop journey passed)
- Capture: `470e2b2c-eb82-4285-826d-7b63c87afda6`
- Capture URL: `/api/v1/captures/secrets-manager/470e2b2c-eb82-4285-826d-7b63c87afda6/file`
- Video: H.264 MP4, 1920x1080, 12.133 seconds, 1,639,347 bytes
- Video checksum: `sha256:f078102d89830f415ebf6716f22e2cde66f0c2083646764b2cb546567934f5bd`
- Manifest: `recording-1786312457358.mp4.manifest.json`, state `passed`, all five required gates passed

The Secrets Manager recording visibly reaches the real dashboard (including
the `Secrets Manager · Dark Chrome Security Dashboard` window title), rather
than stopping at the Electron splash or a blank desktop.

### Certification matrix: platform + BAS in one run

The stronger certification run puts the platform-conformance journey and the
real scenario BAS journey in one required matrix against the same artifact,
target, and profile:

- Matrix run: `run-63a9781eaacc421f49fabd83`
- Matrix gate: passed, 2 required / 2 applicable / 2 passing
- Artifact digest: `sha256:83eb0022a59f76a0dc92eb63271b36da41e8e6d9dabe4a62e30b134c8dd0c781`
- Target: `local-linux-amd64`
- Platform journey capture: `bd268c51-6e78-44b6-aac5-1ff435629e1d` and
  `b0fd9203-9d54-4a09-be75-b3577ef79d5c`
- Platform video: [MP4](/api/v1/captures/secrets-manager/b0fd9203-9d54-4a09-be75-b3577ef79d5c/file),
  H.264 1920x1080, 11.800 seconds, checksum
  `sha256:d2f875450de05e0b0d71dc34eadc5698fac639f33e273e953d26d51804c4ee21`
- BAS workflow run: `5d44569d-4474-4bbd-bc06-15b50714c274`
- BAS video: [MP4](/api/v1/captures/secrets-manager/90c1d5de-0f9d-4441-a270-5967475d9dbe/file),
  H.264 1920x1080, 12.333 seconds, checksum
  `sha256:a614568a5d325c5b337f410387b3b46192ee8cab96434a604aee74430129354a`

Both cells carry target and machine assertions. The BAS cell additionally
carries three checksummed Workflow Health artifacts. Both videos were fetched
from their persisted capture routes and visually inspected; they reach the
real Secrets Manager Electron dashboard.

### Certification matrix: platform + mutating BAS journey

The existing mutating desktop workflow was also rerun through the matrix
executor against a fresh generated `scenario-to-desktop` AppImage:

- Matrix run: `run-ca34229709796ab4ef200722`
- Matrix gate: passed, 2 required / 2 applicable / 2 passing
- Artifact digest: `sha256:9ab0b7764c2b681fbac4ef19843a2f7eaf6d39d90605dd6e8d164e97a540b975`
- BAS workflow run: `84a1d07b-3c2c-46fe-ae75-ceef321da8dd`
- BAS asset: `bas/cases/04-evidence/leased-desktop-evidence.json`
- Platform video: [MP4](/api/v1/captures/scenario-to-desktop/416ae74f-d249-468d-8182-0694a81aec2a/file),
  H.264 1920x1080, 10.067 seconds, checksum
  `sha256:d8e158697222acc10319bd6912e6c0eef19dd621c1fbbb7bcb5b482fba847ce2`
- Mutating BAS video: [MP4](/api/v1/captures/scenario-to-desktop/660ae988-baa8-4466-9e93-5752f2179afe/file),
  H.264 1920x1080, 14.467 seconds, checksum
  `sha256:bc36ac2e7faa1bfcd41389a082f622e785378b6f691ec901b29772a48668405b`
- Cell evidence: three checksummed BAS artifacts, desktop video, target
  identity, and machine assertion; routed isolation validation passed.

The mutating video visibly reaches the generated `Scenario to Desktop`
Electron application. The BAS workflow's durable timeline contains the
leased fixture interaction; the provider's isolation validator rejected any
run without installed, heartbeating, cleared isolation or with primary
storage activity.

### Linked BAS + desktop matrix capture

The final one-cell local Linux matrix run binds provider evidence and the
playable desktop recording in the same passing cell:

- Matrix run: `run-d9ca84cdc491292e9def58ab`
- Matrix gate: passed, 1 required / 1 applicable / 1 passing
- Artifact digest: `sha256:83eb0022a59f76a0dc92eb63271b36da41e8e6d9dabe4a62e30b134c8dd0c781`
- BAS workflow run: `699e87f6-2021-4cda-b760-87147e027050`
- BAS asset: `bas/cases/01-foundation/dashboard-smoke.json`
- Desktop capture: `36df2d6a-47eb-45df-8c37-63cf941370a3`
- Capture URL: `/api/v1/captures/secrets-manager/36df2d6a-47eb-45df-8c37-63cf941370a3/file`
- Video: H.264 MP4, 1920x1080, 18.933 seconds, 2,308,892 bytes
- Video checksum: `sha256:56d335b029208c39319388019838e05767c6b8ef80f30d6c01f41cc8e081c253`
- Cell evidence: three checksummed BAS JSON artifacts, the MP4, target identity,
  and machine assertion; cell disposition `PASS`

The MP4 was fetched from the matrix evidence URI and visually inspected. It
shows the real Secrets Manager Electron window and dashboard, including the
window title, dashboard tabs, credential diagnosis, and keyring report.

## Reviewable capture

- Scenario: `secrets-manager`
- Artifact: `secrets-manager/platforms/electron/dist-electron/Secrets Manager-1.0.0.AppImage`
- Artifact digest: `sha256:45401dabd3e49145ab69a2e6f6eb0d44907bef2c777a2ede4d2372cd39c72bc4`
- Target: `local-linux-xvfb`
- Renderer: `Secrets Manager · Dark Chrome Security Dashboard`
- Transport: loopback-only, authenticated CDP
- BAS asset: `bas/cases/01-foundation/dashboard-smoke.json`
- Workflow Health run: `a7311bcb-4358-4858-beeb-4f3f2b5bf186`
- BAS execution: `0a44433f-446a-4d06-b8c1-7a5d7bb16274`
- BAS result: passed, 1 selected / 1 executed / 1 passed / 0 refused
- Isolation: 0 primary requests, 0 primary writes, 0 test-pool requests, 0 test-root writes
- Desktop capture: `7fef7af1-216b-415d-a17c-e97aee34a95d`
- Capture URL: `/api/v1/captures/secrets-manager/7fef7af1-216b-415d-a17c-e97aee34a95d/file`
- Video: H.264 MP4, 1920x1080, 30 seconds, 2,330,338 bytes
- Video checksum: `sha256:16f6504ce9f63c1c1c3271f568930eb7dbfe8c79a23c549ea9789afb3df17e9c`

The capture is persisted in the scenario-to-desktop capture store and remains
available through the capture URL while the scenario API is running. The
capture metadata records the measured duration and media dimensions; these
values are derived from the decoded MP4 rather than the recorder's command
output.

## Reviewable mutating capture

- Scenario: `scenario-to-desktop`
- Artifact: generated Linux AppImage from pipeline `9b0aaa42-7067-d9e3-8a6c-8f1c8935056e`
- Artifact digest: `sha256:2116bdfdda399d4c3c92c6879a47be9d0e4dfedb4f268b57790671726f164ca4`
- Target: `local-linux-xvfb`
- Renderer: `Scenario to Desktop` at `http://127.0.0.1:22829/`
- Transport: loopback-only, authenticated CDP
- BAS asset: `bas/cases/04-evidence/leased-desktop-evidence.json`
- Workflow Health run: `d7daa72a-63e8-48a8-a6a6-f61407d8a62f`
- BAS execution: `a508201b-4b29-4b44-b49c-6afc0bc2e206`
- BAS result: passed, 1 selected / 1 executed / 1 passed / 0 refused
- Isolation: 0 primary requests, 0 primary writes, 0 test-pool requests; 3 test-root writes
- Desktop capture: `048f1780-3eec-4548-a46b-61468664800d`
- Capture URL: `/api/v1/captures/scenario-to-desktop/048f1780-3eec-4548-a46b-61468664800d/file`
- Video: H.264 MP4, 1920x1080, 44.2 seconds, 1,675,864 bytes
- Video checksum: `sha256:e35adc9fed4aab26a88025cc4a08f2f509449f0384f97702d0ae70245c0ff918`

The video visibly shows the Electron window and the BAS fixture result:
`passed: fixture-desktop.AppImage and fixture-smoke-report.json; leased writes: 3`.

The generated-app launch path was corrected so a validation renderer URL is
used as the app's actual load URL when the live scenario UI port differs from
the build-time default. The recording integrity gate was also corrected to
accept this dark application based on high-contrast content while still
rejecting a uniform dark desktop.

## Recording-path correction

Live-desktop previously persisted a recording without decoding it, which made
a blank/static recording look reviewable and reported `duration_ms: 0` for the
native FFmpeg adapter. `StopRecordingAction` now requires a non-empty video,
decodes and checks the MP4 with `screenrecording.InspectVideo`, rejects missing
useful application frames, and persists the measured media metadata. The
focused action tests cover both valid persistence and invalid-video rejection.
