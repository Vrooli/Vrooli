# Managed release authority

Vrooli owns release-manifest signing through the `vrooli release-authority`
control-plane command. It creates an RSA-3072 keypair in Go, stores the
private half only in the operating system's native credential store through
`internal/credentialauthority.Authority`, and atomically writes the public half to
`install/vrooli-release.pub`. It also synchronizes the public anchor embedded
in both bootstrap installers, so Linux and Windows release verification use
the same authority.

No private-key file, environment variable, CLI argument, or repository entry
is used. The same command surface works on every supported operating system;
it fails closed on a platform whose native credential provider is unavailable.
That is intentional: a plaintext fallback would make release trust weaker than
having no authority at all.

## Lifecycle

```bash
# Creates a key only when no managed key exists. The explicit flag is required
# when replacing the checked-in trust anchor from an earlier authority.
vrooli release-authority init --replace-trust-anchor

# Reveals only public operational state, never private material.
vrooli release-authority status --format json

# Validates every staged byte, then writes release-manifest.sig.json.
vrooli release-authority sign --stage /path/to/staged-release
```

To add a durable test artifact to an evidence release, use the control plane
rather than copying it into the stage manually. It accepts only a regular
source file and a safe stage-local name, calculates the digest, and rewrites
the canonical manifest. That necessarily invalidates the previous signature,
so sign the completed stage afterward.

```bash
vrooli release-authority add-evidence \
  --stage /path/to/evidence-release \
  --source /path/to/bas-timeline.json \
  --name scenario-console-bas-timeline.json \
  --role bas-run \
  --provenance "BAS run <execution-id>" \
  --os linux --arch amd64
vrooli release-authority sign --stage /path/to/evidence-release --overwrite
```

`init` is idempotent after the authority exists. `regenerate` is deliberately
different: it replaces both the private key and `install/vrooli-release.pub`.
Use it only for a deliberate trust-root reset:

```bash
vrooli release-authority regenerate --replace-trust-anchor
```

Replacing the trust anchor means artifacts signed by the previous key no
longer verify against the new anchor. Existing released clients must therefore
be upgraded through a trusted transition before a production rotation.

## Release signing versus installer signing

Release-manifest signing authorizes the exact resource, tool, installer, and
evidence bytes that Vrooli packages. It is independent of operating-system
installer signing. Windows Authenticode, macOS Developer ID/notarization, and
Linux package signatures remain publisher-facing platform mechanisms.

## Verification

```bash
vrooli-dist --verify-release-manifest \
  --root /path/to/Vrooli \
  --release-artifact-root /path/to/staged-release \
  --trust-mode production
```

Production verification requires the detached signature created by the managed
authority. `development-local` still verifies every listed byte but is
explicitly non-promotable.
