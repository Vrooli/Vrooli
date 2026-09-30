// Package fetch is the shared download/checksum/artifact-validation spine for
// every catalog in image-tools that must put bytes on disk — today the model
// registry (internal/models), tomorrow the adapter registry (internal/adapters).
//
// Why this boundary exists: installing a model and installing a LoRA / ControlNet
// / IP-Adapter are the SAME mechanical problem — resolve a download spec (direct
// asset URLs or a pinned HuggingFace repo snapshot), stream the bytes, reject a
// landing PAGE masquerading as a weight, pin/verify a checksum, and refuse to
// start when the disk is too full. Only the *catalog vocabulary* (what a Model vs
// an Adapter is, which ops it serves, how it is governed) differs. Keeping the
// mechanics here means both catalogs share one audited, tested install path
// instead of forking it — and a fix to the install-stub guard (binaryfetch) or
// the tree-manifest hash lands once.
//
// The package owns the download *spec* types (Asset, RepoSpec, ArtifactKind) and
// the *I/O seams* (Downloader, RepoFetcher) plus their production implementations
// (HTTPDownloader, HFSnapshotFetcher) and the integrity helpers (ValidateArtifact,
// HashFile, TreeManifestHash) and free-space probe (DefaultDiskAvail). It knows
// nothing about operations, architectures, tiers, or SQLite — those are catalog
// concerns. internal/models re-exports the spec types (Asset/RepoSource/
// ArtifactKind) as its own catalog vocabulary so the seed schema is unchanged.
package fetch
