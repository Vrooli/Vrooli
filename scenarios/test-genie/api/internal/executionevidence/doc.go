// Package executionevidence owns the durable, versioned evidence emitted by a
// Test Genie execution. It is the sole home for manifest validation, immutable
// artifact references, integrity digests, and bounded artifact writes.
//
// The package deliberately does not know orchestration, HTTP, SQLite, or phase
// providers. Those layers exchange compact summaries and ArtifactRef values;
// only this domain owns detailed evidence bytes.
package executionevidence
