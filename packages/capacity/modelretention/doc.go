// Package modelretention is the shared contract between a model store's
// request boundary (the activity edge), its owner CLI (the one fenced
// eviction operation) and storage-manager (which decides what to evict).
//
//   - usage.go: per-model last-request timestamps recorded from real traffic
//     at the activity edge. Residency (/api/ps) is not use; only a request
//     that names a model counts. Each model also splits its requests by the
//     caller the edge identified, so an owner can see who uses a model.
//   - fence.go: a cross-process per-model fence built on advisory file
//     locks. Every request holds a shared lock on its model for the length of
//     the request; eviction takes the exclusive lock without waiting, so it
//     refuses while a request is in flight and no request can start while it
//     deletes. The kernel drops the locks when a process exits.
//   - receipts.go: the reacquisition receipt an owner writes before it
//     deletes, so getting a model back is one command and its cost is known.
//   - manifests.go: unique and shared blob bytes from the store's manifests,
//     so a ranking counts what deleting a tag actually frees.
//   - inventory.go: the document an owner's `models retention --json` prints.
package modelretention
