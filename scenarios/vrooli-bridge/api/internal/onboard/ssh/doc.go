// Package ssh gives the bridge control plane a bounded, owner-credentialed
// "first touch" on a fresh node: with a transient owner-supplied
// {host, user, password} it establishes working passwordless key-based SSH,
// then discards the password.
//
// It is a deliberate DUPLICATE of scenarios/scenario-to-cloud/api/ssh — the
// proven in-repo precedent for key generation, TOFU (trust-on-first-use)
// host-key handling, password-authenticated key copy, remote command
// execution, and SSH error classification — adapted so all key material lives
// under the bridge-owned state directory instead of the operator's ~/.ssh. Per
// the repo's duplicate-before-extract rule this is intentionally NOT a shared
// package: the two callers have different state-path and lifecycle needs, and a
// premature shared abstraction would couple them.
//
// Credential discipline (DECISIONS.md, "managed first touch"): the owner's
// password is held in memory only for the single key-copy dial and zeroed
// immediately afterward; it is never written to disk, logs, or the database.
// The generated keypair and known_hosts persist 0600 (dir 0700) under the
// bridge state dir, and every subsequent node interaction is key-based.
package ssh
