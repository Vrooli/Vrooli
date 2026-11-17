// Package volumeremediation owns safe inspection and repair orchestration for storage volumes.
// Its identity-bound operation lease coordinates cooperating control-plane
// processes only; it cannot prevent a physical unplug or an operation that
// bypasses the supported host owner path.
package volumeremediation
