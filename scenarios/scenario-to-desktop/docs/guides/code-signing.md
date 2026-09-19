# Code Signing Guide (Windows, macOS, Linux)

This guide explains what must be signed for each platform, which OS can perform the signing, how to obtain certificates/keys, and how to configure signing in scenario-to-desktop.

## Before installer signing: release trust

Release trust is a separate, earlier gate for bundled resource bytes. It does
not affect whether Windows, macOS, or Linux recognizes an installer publisher.
For a local evidence build, provide a verified staged artifact root and choose
`--artifact-trust-mode development-local`; the resulting bundle is visibly
non-promotable and deployment/publishing is refused. Production instead uses
`--artifact-trust-mode production`, which requires a valid detached
`release-manifest.sig.json` from the project-managed Vrooli release authority
before packaging begins. `vrooli release-authority` retains that authority's
private key in the native credential store; see
[`docs/configuration/release-authority.md`](../../../../docs/configuration/release-authority.md).
Neither mode creates or requires an OS code-signing key.

## Platform Summary (What to Sign, Where to Sign, Cost/Keys, Reuse)

| Platform | What is signed | Where you must sign | Keys/certs & cost | Can reuse across scenarios? |
|----------|----------------|---------------------|-------------------|-----------------------------|
| Windows | Executables + installers (Authenticode) with timestamp | Windows host using SignTool (part of Windows SDK/VS). EV tokens require Windows. | Commercial code-signing cert (PFX or hardware token). ~$200–$400/yr (OV), ~$400–$700/yr (EV). Identity/business verification; issuance can take 1–3+ business days. | Yes. One org cert can sign many apps/installers. |
| macOS | .app bundles + installers; notarization stapling | macOS host with `codesign` + `notarytool` (Xcode CLT). | Apple Developer Program ($99/yr) + Developer ID Application cert. Notarization uses App Store Connect API key. Approval may take hours–days (account + D‑U‑N‑S if org). | Yes. One Developer ID cert/API key per team can sign/notarize many apps. |
| Linux | Packages (DEB/RPM) and AppImage via GPG signatures | Linux host with `gpg` + package signers (`dpkg-sig` on Debian/Ubuntu; `rpm`/`rpm-sign` for rpmsign). | GPG key you generate yourself (free). Immediate use. | Yes. Same key can sign many packages. |

**Can I sign everything from one machine?**  
- Windows signing: realistically Windows only (SignTool + optional EV token).  
- macOS signing/notarization: macOS only (needs `codesign`/`notarytool` and macOS keychain).  
- Linux signing: Linux (or WSL) with GPG + packaging tools.  
You can build unsigned elsewhere and sign on the target OS if needed.

**Do keys have to be created on that OS?**  
- Windows: CA issues a PFX or hardware token; importable elsewhere, but signing still runs on Windows.  
- macOS: Developer ID cert is issued into a macOS keychain; exportable (.p12) but still must sign/notarize on macOS.  
- Linux: GPG keys can be generated anywhere and copied.

**Paying and approval timing**  
- Windows CA certs: paid, identity vetting before issuance.  
- Apple Developer ID: $99/yr; need approved developer account before generating cert and API key.  
- Linux GPG: free, instant.

## Workflow Overview
- Decide platforms and formats (see `cross-platform-builds.md` for build matrix).
- Acquire the required cert/key per platform.
- Install platform tools on the OS that will perform signing.
- Configure signing in scenario-to-desktop (UI Signing tab or `scenario-to-desktop signing set`).
- Run **Validate**, then build with signing enabled; notarize for macOS if required.

## Signing Tools Panel (what it checks)
- Runs local detection to confirm the OS-level CLIs are available before you try to sign:
  - `signtool` (Windows SDK/VS), `osslsigncode` (Linux/macOS alternative for Windows EXE signing),
  - `codesign`, `notarytool`, `altool` (macOS),
  - `gpg`, `rpmsign` (via `rpm` or `rpm-sign`), `dpkg-sig` (Linux package signing).
- Shows per-platform cards with status, version, and resolved path when found.
- When missing, shows remediation text plus quick install commands (e.g., `xcode-select --install`, `apt install osslsigncode`, `brew install gnupg`).
- If a tool errors, the panel surfaces the error text so you can fix PATH or reinstall.
- Refresh by reopening the Signing tab or revisiting the page after installing tools.

## Windows (Authenticode)
**What you need**
- Code-signing certificate (`.pfx/.p12` + password) or a thumbprint for a cert in the Windows certificate store.
- Timestamp server URL (defaults provided).

**Install tools**
- SignTool (comes with Windows 10/11 SDK or Visual Studio; install “Windows SDK” via VS installer).

**How to list certificates**
```powershell
# List code-signing certs (CurrentUser\My store)
Get-ChildItem -Path Cert:\CurrentUser\My | Where-Object { $_.EnhancedKeyUsageList -match "Code Signing" } |
  Select-Object Subject, Thumbprint, NotAfter
```

**Save config**
- In Signing tab, choose **Windows → Store** and paste the thumbprint, or choose **File** and set `certificate_file` + password env var (e.g., `WIN_CERT_PASSWORD`).

**Reference**
- Microsoft: Code signing certificates and SignTool — https://learn.microsoft.com/windows/win32/seccrypto/signtool

## macOS (Developer ID + Notarization)
**What you need**
- Developer ID Application certificate in your login keychain (from Apple Developer account).
- Team ID (10 characters).
- Optional notarization credentials (recommended): App Store Connect API key (Key ID, Issuer ID, `AuthKey_XXXX.p8`) or Apple ID + app-specific password.

**Install tools**
- Xcode Command Line Tools: `xcode-select --install`

**How to list identities**
```bash
security find-identity -v -p codesigning
```

**Notarization (API key, recommended)**
- Create an App Store Connect API key, download `AuthKey_XXXX.p8`, note **Key ID** and **Issuer ID**.
- In Signing tab, enable notarization and fill Key ID, Issuer ID, and key file path (or set `APPLE_API_KEY_ID`, `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_FILE` env vars).

**Reference**
- Apple Developer ID & notarization — https://developer.apple.com/support/developer-id
- Xcode notarization docs — https://developer.apple.com/documentation/xcode/notarizing_macos_software_before_distribution

## Linux (GPG for .deb/.rpm/AppImage)
**What you need**
- A GPG private key suitable for signing packages (key ID or fingerprint).
- Optional custom keyring paths if not using the default.

**Install tools**
- Debian/Ubuntu: `sudo apt update && sudo apt install gnupg rpm osslsigncode` (rpmsign is provided by rpm); optionally `sudo apt install dpkg-sig` if available/enabled (universe)
- RHEL/CentOS/Fedora: `sudo dnf install gnupg2 rpm-sign` (rpmsign), and add `osslsigncode` if you need Windows EXE signing from Linux

**Managed key custody (recommended)**

A generated key is custodied by the native credential authority, not written as
a keyring into the repository. One publisher key can sign every desktop app:
name a shared logical identity (for example `vrooli/desktop-signing`) and every
scenario that names it uses the same key. `generate-key` is idempotent per
identity — the first run mints the key, and later runs for other scenarios reuse
the existing key instead of rotating it.

```bash
# First scenario: mint the shared publisher key. The private key and a fresh
# random passphrase are stored in the credential authority; only the public key
# and fingerprint touch the scenario.
scenario-to-desktop signing generate-key <first-scenario> \
  --name "Your Publisher" --email publisher@example.test \
  --logical-id vrooli/desktop-signing

# Later scenarios: same command and identity. The existing key is reused, so
# every scenario signs with the same publisher fingerprint.
scenario-to-desktop signing generate-key <second-scenario> \
  --name "Your Publisher" --email publisher@example.test \
  --logical-id vrooli/desktop-signing

scenario-to-desktop signing validate <first-scenario>
scenario-to-desktop signing ready <first-scenario>
```

When `--logical-id` is omitted, custody defaults to the scenario's own
namespace, `vrooli/scenario-to-desktop/<scenario>`, which is a private key per
scenario. Pass `--force` only to rotate a key deliberately; rotation creates a
new fingerprint, so every other scenario that names the identity must update its
`gpg_key_id` before it can sign again.

**Automatic signing (default)**

Every desktop app generated from a scenario is signed with the shared publisher
key automatically when it is available. During generation, when the scenario has
no `signing.json`, `scenario-to-desktop` resolves `vrooli/desktop-signing` from
the credential authority, writes the scenario's `signing.json`, and wires the
Linux artifact-signer hook. No per-scenario command is required.

- An existing `signing.json` is respected unchanged.
- An explicit `signing.json` with `"enabled": false` is an opt-out; it is not
  overridden.
- If the shared key is absent or the authority is unavailable, generation
  proceeds unsigned and records the reason in the build log. Provision the key
  with `scenario-to-desktop signing generate-key <scenario> --name <publisher>
  --email <email> --logical-id vrooli/desktop-signing` and regenerate.

The hook and switch are written at generation time, so an app whose desktop
project predates automatic signing must be regenerated once to pick it up.

The generated `signing.json` records a `managed_key` block:

```json
{
  "enabled": true,
  "linux": {
    "gpg_key_id": "FINGERPRINT",
    "gpg_passphrase_env": "VROOLI_GPG_PASSPHRASE",
    "managed_key": { "logical_id": "vrooli/desktop-signing" }
  }
}
```

The authority holds the ASCII-armored private key in field `gpg-private-key`
and the passphrase in `gpg-passphrase`. Inspect them through the control plane;
the values are never displayed:

```bash
vrooli credentials status --identity vrooli/desktop-signing --field gpg-private-key --format json
vrooli credentials status --identity vrooli/desktop-signing --field gpg-passphrase --format json
```

At build time the build resolves both fields, imports the key into an ephemeral
0700 GPG home, and injects `VROOLI_GPG_HOMEDIR` plus the passphrase environment
variable only for the signing command. Neither value is written to the generated
project, the process environment at large, or the metadata sidecar. The public
key is published to `scenarios/<scenario>/signing/public-key.asc`.

**External keyring (optional)**

To sign with a keyring you manage yourself, set `gpg_homedir` and
`gpg_passphrase_env` instead of `managed_key`. The build then reads the ambient
passphrase variable and the supplied keyring, exactly as before.

```bash
gpg --list-secret-keys
```

The generated Linux build installs an electron-builder `afterAllArtifactBuild`
hook. It creates an ASCII-armored detached `.asc` signature for each Linux
artifact and emits `linux-update-metadata.json` beside the artifacts. That
metadata binds the package filename, platform, detected architecture, release
version, update channel, artifact SHA-512 digest, and signature SHA-512 digest.
The passphrase is read only from the configured environment variable; it is
never written to the generated project or metadata. The publication owner must
serve the artifact, signature, and metadata from the same immutable release
revision. This is the Linux trust boundary; GPG does not provide a universal
OS-level trust prompt equivalent to macOS notarization or Windows Authenticode.

**Reference**
- Debian package signing — https://wiki.debian.org/Packaging/Signing
- RPM signing — https://rpm-packaging-guide.github.io/#signing

## If You Don’t Have Certificates Yet
- You can ship unsigned installers for local testing; OS will warn users.
- For sandbox-only tests:
  - Windows: generate a PFX with OpenSSL; set `certificate_file` + password env var.
  - Linux: create a local GPG key with `gpg --quick-generate-key`.
  - macOS: you need a real Developer ID cert for Gatekeeper trust (self-signed won’t satisfy users).

## Troubleshooting
- **“Tool not found”**: Install platform CLI (signtool, codesign/notarytool, gpg/rpmsign/dpkg-sig) then click Refresh in Signing tab.
- **“Certificate not valid for code signing”**: Ensure EKU includes Code Signing; try another cert or reissue.
- **Notarization fails**: Confirm API key file path and permissions; try `xcrun notarytool history --key <file> --key-id <id> --issuer <issuer>`.
- **Expired/expiring cert**: Replace before publishing; expiry warnings appear in Signing and Generator flows.
