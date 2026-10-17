// Package recoverypoint is the engine-independent owner of cloud recovery
// points: consistent, encrypted, checksummed captures of a deployment's
// declared persistent-data bindings that a replacement host can restore and
// verify without the host that produced them.
//
// It is shared by the target-local owner (internal/cloudtarget, exposed as
// `vrooli cloud-target data backup|restore|verify`) and the scenario-to-cloud
// API (api/backup), so both sides read and write exactly one on-disk format.
//
// Layout of one recovery point directory:
//
//	recovery-point.json           manifest: bindings, refs, checksums, digest
//	<binding-id>.sealed           AES-256-GCM envelope over the provider artifact
//
// Rules the package enforces:
//
//   - A database binding is captured with database-native consistency through
//     its provider (pg_dump custom format, VACUUM INTO, ...); a generic file
//     copy of a live database is never accepted as a backup.
//   - Every artifact is sealed under a key resolved from a reference; key
//     material is never written into the manifest or a receipt.
//   - A manifest whose digest or artifact checksums do not verify is corrupt
//     and restoration refuses before any target is touched.
//   - A missing key is a typed blocker naming the key reference, never a
//     silent skip.
//   - Restore writes only into targets a provider has proven clean; a failure
//     never reports success for the bindings it did write.
package recoverypoint
