# Crypto Infrastructure CI2 evaluator consumer

This is the first importing consumer for `github.com/vrooli/chain-forensics`.
It evaluates the package's typed `Case` contract, invokes its canonical
validation and private packet serializer, and reports chain coverage, flow
grades, legal-claim boundaries and residual risks. It intentionally contains
no duplicate case, occurrence, edge, source or packet models.

From this directory, `go test ./...` exercises the evaluator against the shared
package's bundled public case. `go run ./cmd/ci2-evaluate` emits a deterministic
consumer report. The local `replace` keeps this monorepo checkout runnable;
release builds should resolve a tagged `github.com/vrooli/chain-forensics`
version and remove that development replacement.

The report is evidence coverage, not an anonymity score or a claim that a
system is decentralized. CI2's full mission still requires per-layer observer
and control inventory, adversary models, linkage attacks and three deployment
proving targets.
