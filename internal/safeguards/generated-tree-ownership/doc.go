// Package generatedtreeownership hands root-owned generated Proto trees in the
// repository back to the operator.
//
// A root-run materialization (for example `sudo vrooli setup`) can leave
// packages/proto/gen or its replaced copy gen.previous owned by root. The
// operator can write the directory but not delete or replace those files, so the
// next unprivileged `make build` fails with "permission denied" while
// publishing the compatibility view. Setup is the only place sudo runs, so
// setup is where this is repaired. It changes ownership and nothing else, and
// never follows a symlink.
package generatedtreeownership
