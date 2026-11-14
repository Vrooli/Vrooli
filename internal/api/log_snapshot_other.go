//go:build !unix

package api

// Unix FIFO open semantics do not apply on these platforms.
const logSnapshotOpenFlags = 0
