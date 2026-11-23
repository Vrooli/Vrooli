# Fresh maintainer receipt: integration HTTP client

## Finding

`Environment` exposes an `HTTP *http.Client` and the JSON helpers use it, but `Rasterize` and `Composite` reach `postMultipart`, which sends through `http.DefaultClient` instead. An integration caller's injected transport, authentication, routing, and timeout settings therefore do not govern multipart calls. In the regression probe, the configured client can service the request locally, but the current implementation attempts DNS for the scenario URL and fails.

Expected behavior: multipart calls honor the `Environment.HTTP` client, consistently with `getJSON` and `postJSON`.

Operator impact: deterministic and controlled integration runs may fail, bypass transport configuration, or behave differently on hosts whose default network access differs from the configured client.

## Evidence

Added `http_client_test.go`, focused tests using a custom `RoundTripper` to verify both `Rasterize` and `Composite` send through the injected client and return its response bytes. The owner fix routes the shared multipart helper through `Environment.HTTP`, matching the JSON helpers.

Baseline against the frozen package source:

```text
go test -tags=integration ./integration -run '^TestRasterizeUsesEnvironmentHTTPClient$' -count=1
--- FAIL: TestRasterizeUsesEnvironmentHTTPClient
    http_client_test.go:49:
        Error: Received unexpected error:
            Post "http://image-tools.invalid/api/v1/ops/convert?output=bytes": dial tcp: lookup image-tools.invalid on 127.0.0.53:53: no such host
FAIL
```

Overlay proof (temporary overlay changed only `http.DefaultClient.Do(req)` to `e.HTTP.Do(req)` in `lane.go`; the tracked source was not edited):

```text
go test -tags=integration -overlay=integration/.fresh_overlay/overlay.json ./integration -run '^TestRasterizeUsesEnvironmentHTTPClient$' -count=1
ok   backdrop-studio/integration  0.009s
```

The temporary overlay was removed after the run. After repair, both focused cases pass once, five times, and under race five times; `go vet -tags=integration ./integration` passes.

## Limits

The ordinary real-service integration suite was not run because the installed Backdrop Studio API reports a stale source fingerprint for this working tree. `lane.go` itself is behind the `integration` build tag, so the isolated test must carry the same tag; a normal `go test ./integration` has no buildable source and is not a supported owner check. The full Test Genie Unit phase therefore does not execute this integration-only owner boundary. These focused tagged tests prove the client-routing contract but do not exercise image-tools' live multipart parser or rendering behavior.
