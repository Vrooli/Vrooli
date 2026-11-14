# Delivery ramp spine

`github.com/vrooli/vrooli/packages/delivery-ramp-go` is the provider-neutral
contract shared by delivery ramps. It owns target inventory, journey and
evidence schemas, fail-closed dispositions, reference-only deployment
verdicts, and the immutable validation matrix.

A ramp implements only the exported `Prober`, `Builder`, `Driver`, and
`Distributor` interfaces. Platform processes, display systems, capture tools,
credentials, and artifact bytes remain inside the ramp. The matrix reaches a
target through one transport seam; local and bridge transports must preserve
the same target-owned evidence rule.

The named conformance test in
`validationmatrix/reference_ramp_test.go` is the template for future ramps. It
uses exported seams only and covers inventory, build, drive, distribution,
matrix lifecycle, rerun immutability, cancellation, and host/emulator/bridge
reference verdicts.

Run the package checks with:

```sh
GOWORK=off go test ./...
go vet ./...
golangci-lint run --timeout=5m
```
