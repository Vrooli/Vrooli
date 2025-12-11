# Code Signing Guide (Windows, macOS, Linux)

This guide explains what must be signed for each platform, which OS can perform the signing, how to obtain certificates/keys, and how to configure signing in scenario-to-desktop.

## Platform Summary (What to Sign, Where to Sign, Cost/Keys, Reuse)

| Platform | What is signed | Where you must sign | Keys/certs & cost | Can reuse across scenarios? |
|----------|----------------|---------------------|-------------------|-----------------------------|
| Windows | Executables + installers (Authenticode) with timestamp | Windows host using SignTool (part of Windows SDK/VS). EV tokens require Windows. | Commercial code-signing cert (PFX or hardware token). ~$200–$400/yr (OV), ~$400–$700/yr (EV). Identity/business verification; issuance can take 1–3+ business days. | Yes. One org cert can sign many apps/installers. |
| macOS | .app bundles + installers; notarization stapling | macOS host with `codesign` + `notarytool` (Xcode CLT). | Apple Developer Program ($99/yr) + Developer ID Application cert. Notarization uses App Store Connect API key. Approval may take hours–days (account + D‑U‑N‑S if org). | Yes. One Developer ID cert/API key per team can sign/notarize many apps. |
| Linux | Packages (DEB/RPM) and AppImage via GPG signatures | Linux host with `gpg` + package signers (`dpkg-sig`/`rpm-sign`). | GPG key you generate yourself (free). Immediate use. | Yes. Same key can sign many packages. |

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
- Decide platforms and formats (see `CROSS_PLATFORM_BUILDS.md` for build matrix).
- Acquire the required cert/key per platform.
- Install platform tools on the OS that will perform signing.
- Configure signing in scenario-to-desktop (UI Signing tab or `scenario-to-desktop signing set`).
- Run **Validate**, then build with signing enabled; notarize for macOS if required.

## Signing Tools Panel (what it checks)
- Runs local detection to confirm the OS-level CLIs are available before you try to sign:
  - `signtool` (Windows SDK/VS), `osslsigncode` (Linux/macOS alternative for Windows EXE signing),
  - `codesign`, `notarytool`, `altool` (macOS),
  - `gpg`, `rpmsign`, `dpkg-sig` (Linux package signing).
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
- Debian/Ubuntu: `sudo apt install gnupg dpkg-sig`
- RHEL/CentOS/Fedora: `sudo yum install gnupg rpm-sign`

**How to list keys**
```bash
gpg --list-secret-keys
```

**Save config**
- In Signing tab, enable Linux and paste the key ID/fingerprint. Add `gpg_passphrase_env` if your key is protected.

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
