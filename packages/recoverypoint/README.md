# recoverypoint

Engine-independent owner of cloud recovery points: consistent, encrypted, checksummed
captures of a deployment's declared persistent-data bindings that a replacement host can
restore and verify on its own.

The target-local owner (`vrooli cloud-target data backup|restore|verify`) and the
scenario-to-cloud API share this package, so both read and write one on-disk format.
Database bindings are captured with database-native consistency; a file copy of a live
database is never accepted as a backup.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
