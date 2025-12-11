# Code Signing Guide (Windows, macOS, Linux)

This guide gives step-by-step instructions for the minimum viable code signing setup per platform, with links to official references.

## Quick Checklist
- Decide which platforms you will ship.
- Gather the minimum inputs (below).
- Install the platform tools.
- Save config in scenario-to-desktop (Signing tab or `scenario-to-desktop signing set`).
- Run **Validate**, then build with signing enabled.

## Windows (Authenticode)
**What you need**
- A code-signing certificate (`.pfx/.p12`) and its password, or a thumbprint for a cert in the Windows certificate store.
- Timestamp server URL (defaults: DigiCert/Sectigo/GlobalSign).

**Install tools**
- SignTool ships with the Windows 10/11 SDK or Visual Studio. Install “Windows SDK” via the VS installer.

**How to list certificates**
```powershell
# In PowerShell, list code-signing certs (CurrentUser\My store)
Get-ChildItem -Path Cert:\CurrentUser\My | Where-Object { $_.EnhancedKeyUsageList -match "Code Signing" } |
  Select-Object Subject, Thumbprint, NotAfter
```

**Save config**
- In the Signing tab, choose **Windows → Store** and paste the thumbprint, or choose **File** and set `certificate_file` + password env var (e.g., `WIN_CERT_PASSWORD`).

**Reference**
- Microsoft: Code signing certificates and SignTool — https://learn.microsoft.com/windows/win32/seccrypto/signtool

## macOS (Developer ID + Notarization)
**What you need**
- Developer ID Application certificate in your login keychain.
- Team ID (10 characters).
- Optional notarization credentials (App Store Connect API key or Apple ID + app-specific password).

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
- For sandbox testing only:
  - Windows: generate a PFX with OpenSSL, set `certificate_file` + password env var.
  - Linux: create a local GPG key with `gpg --quick-generate-key`.
  - macOS: you must use a real Developer ID certificate for Gatekeeper trust.

## Troubleshooting
- **“Tool not found”**: Install platform CLI (signtool, codesign/notarytool, gpg/rpmsign/dpkg-sig) then click Refresh in Signing tab.
- **“Certificate not valid for code signing”**: Ensure EKU includes Code Signing; try another cert or reissue.
- **Notarization fails**: Confirm API key file path and permissions; try `xcrun notarytool history --key <file> --key-id <id> --issuer <issuer>`.
- **Expired/expiring cert**: Replace before publishing; expiry warnings appear in Signing and Generator flows.
