# Credential escrow integration contract

The control plane owns credential recovery material. Consumers must not read
the credential authority, assemble a recovery bundle, or copy one themselves.
Register `credentialescrow.NewProviders(root, home)` in the
`operatorcapability.Registry`, then use the registry's normal `Discover`,
`Preview`, `Apply`, and `Verify` lifecycle for capability
`credential-escrow`.

`Discover` is metadata-only. It inventories configured addresses through
`credentialinventory.Collect(root)`, rejects incomplete required operator
credentials, and returns approved storage candidates and the missing typed
inputs. Candidate selection is always an operator input (`sink`); neither the
provider nor a consumer selects or provisions an off-host location. The
`recovery_passphrase` is a secret input used in memory. Consumers must not
persist it or include it in logs, requests, or receipts.

`Apply` re-discovers/revalidates the selected destination, collects the
authoritative credential inventory, exports the bundle through
`credentialauthority`, reads it back with the supplied passphrase, and writes
metadata-only evidence. A successful result includes `recovery-bundle`
evidence with artifact identity, source generation, checksum, verification
time, and address coverage; receipt coverage contains logical identity/field
addresses only, never values. `Verify` rechecks freshness and coverage and
binds those receipts to the caller's verification context.

## Data Backup Manager seam

DBM should consume the `credential-escrow` capability through the existing
control-plane capability registry and its operator-reviewed apply flow. The
result's verified `recovery-bundle` evidence is the dependency DBM can use to
gate clean-node rehearsal readiness. DBM needs no credential-authority API and
must not treat a local/removable candidate as off-host: the selected
destination's host topology is a separate proof. A fleet-node/SFTP candidate
is not currently exposed by this provider; adding that candidate belongs to
the control-plane destination adapter/owner path. Do not extend DBM destination
schemas or resource-kopia SFTP/S3 backends to move this bundle.

The current provider supports discovered local/removable candidates and an
operator-supplied S3-compatible URI. S3 artifact upload verifies the remote
object by read-back inside `securestore.UploadS3Artifact`; the control-plane
provider owns the copy and receipt. The provider does not create an off-host
copy until an operator supplies and confirms the selected sink through the
capability flow.
