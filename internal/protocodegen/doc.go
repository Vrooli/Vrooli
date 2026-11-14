// Package protocodegen holds invariants for the proto codegen pipeline
// in packages/proto. The pipeline itself runs via `buf` and produces code
// in packages/proto/gen/; this package only contains test-time guards
// (e.g. CD-1: no BSR remote plugins) so the rules live in CI.
package protocodegen
