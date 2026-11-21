// Package gocache owns the host's coordination protocol for the shared Go
// build cache (GOCACHE).
//
// # Why a protocol is needed
//
// cmd/go promises that after a cache Get or Put the output file stays on disk
// until the go command closes its cache at exit. Compile, link and vet read
// outputs long after the lookup and without continuously open handles, so
// neither file age, cache size, nor an open-handle snapshot proves that an
// output is unused. Deleting cache files underneath live builds breaks them.
// Go's only built-in lifecycle is a five-day age trim, which never bounds a
// cache that churns tens of gigabytes a day.
//
// # The protocol
//
// Every go command on the host is pointed, through the user's go env file
// (go env -w GOCACHEPROG=...), at vrooli-gocacheprog: a GOCACHEPROG helper that
// serves the same on-disk store and layout cmd/go itself uses. Each helper
// process is one cache session, living exactly as long as its go command.
//
//   - A session records every action and output ID it hands out in an
//     append-only pin journal, and holds an exclusive lock on that journal for
//     its lifetime. The kernel releases the lock when the session ends, however
//     it ends, so a journal whose lock can be taken belongs to a dead session.
//   - A lookup and its pin happen under a shared coordination lock. The
//     reclaimer deletes only while holding that lock exclusively, after
//     re-reading every live journal, so no output can be handed out between
//     the reclaimer's pin check and its deletion.
//   - Before each deletion batch the reclaimer audits the host's live go
//     commands. A go command that uses this store without a session (started
//     before activation, GOENV=off, an older toolchain, a different
//     GOCACHEPROG) makes reclaim refuse rather than guess.
//   - Recency is a second, independent guard: entries used within the
//     minimum idle window are never eligible, which covers consumers such as
//     go/packages that read export data after `go list -export` exits.
//
// Reclaim is size-targeted least-recently-used deletion bound to one store by
// a plan token. The plan names the store identity and the last-use cutoff;
// apply refuses a token whose store no longer matches.
//
// The protocol cannot see go invocations that bypass the user's go env file
// and also run without a session; the audit refuses reclaim while any such
// process is alive, but it cannot see one that starts and finishes inside a
// single deletion batch. `go clean -cache` remains uncoordinated by design of
// cmd/go and must not be run on a shared host.
package gocache
