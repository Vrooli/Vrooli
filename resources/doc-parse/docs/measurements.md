# Portable parser measurements

This is the Phase 1 evidence for the selected portable-runtime candidate. The
same JSONL request corpus was run through the release native Rust shim and the
same Rust code compiled to `wasm32-wasip1` and executed with Wazero.

The harness compares normalized JSON output byte-for-byte and records wall time
and observed process RSS. Raw rows are retained in
[`measurements.raw.csv`](measurements.raw.csv).

## Corpus

The corpus contains 42 fixtures: four each of DOCX, ODT, EPUB, PDF, RTF, CSV,
XLSX, PPTX, and password-protected DOCX, plus malformed PDF and scanned-PDF
cases. The password for protected fixtures is intentionally not supplied; the
expected result is a terminal password-required error.

## Results

Times are milliseconds per fixture. RSS is the process-level observed value,
not an isolated parser allocation measurement.

| Option | Input class | Fixtures | Median wall | P95 wall | Output equality |
| --- | --- | ---: | ---: | ---: | --- |
| Native | documents | 28 | 1 ms | 2 ms | pass |
| Native | PDFs | 9 | 5 ms | 15 ms | pass |
| Native | terminal/error cases | 5 | 1 ms | 2 ms | pass |
| WASI + Wazero | documents | 28 | 4 ms | 5 ms | pass |
| WASI + Wazero | PDFs | 9 | 4 ms | 8 ms | pass |
| WASI + Wazero | terminal/error cases | 5 | 3 ms | 4 ms | pass |

The harness emitted: `equality,all,42 fixtures matched byte-for-byte`.
Both options are below the plan's 250 ms tier-1 PDF p95 gate. WASI is selected
for the first production resource surface because it is one artifact for every
target pair and runs in-process through the pure-Go Wazero runtime; native
artifacts remain the declared fallback if a target cannot execute WASI.

## Reproduction

The resource shim and WASI module are built in the Rust build lane, since this
host does not provide Rust natively:

```text
docker run --rm -v /home/matthalloran8/Vrooli:/workspace -w /workspace/resources/doc-parse rust:latest cargo build --release
docker run --rm -v /home/matthalloran8/Vrooli:/workspace -w /workspace/resources/doc-parse rust:latest /bin/sh -lc 'rustup target add wasm32-wasip1 && cargo build --release --target wasm32-wasip1'
```

Then build and run the Go harness from `cli/` with the native binary and WASI
module paths. A successful run must exit zero and emit the equality row above.
