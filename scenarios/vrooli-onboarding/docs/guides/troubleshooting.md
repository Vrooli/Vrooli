# Troubleshooting

If the UI shows no scenarios, verify repository manifests are available to the
scenario process. If credentials are unconfigured, provision the exact declared
logical identity through onboarding or `vrooli credentials provision` on stdin.

A missing or unreadable credential never blocks a scenario from starting. The
scenario runs and the resources that declare the credential report unhealthy
until it is provisioned, so onboarding can be completed after the fact without
restarting anything. Run `vrooli credentials doctor` to see whether the cause is
an unset value, an unreachable secure store, or a host with no backend — each
has a different fix, and the command names it. Plaintext files and Vault
fallback reads remain unsupported on every platform.

Onboarding works on a host with no desktop session. Every native credential
store — libsecret on Linux, the macOS Keychain, the Windows Credential
Manager — needs a logged-in graphical session, so a server, a CI runner, or a
Raspberry Pi has none of them. There, Vrooli uses an **encrypted file store**
instead, and `doctor` reports `Backend: encrypted-file` with the key wrap
holding it open. Create it once before provisioning anything:

```bash
vrooli credentials store init          # a reachable TPM needs no passphrase
printf '%s' "$PASSPHRASE" | vrooli credentials store init   # otherwise
vrooli credentials store status        # names the wraps and what protects them
```

A host with a TPM reaches a working state after a reboot with no human action.
A host without one needs `vrooli credentials store unlock` once per login
session; later commands in that session do not ask again, and
`vrooli credentials store lock` ends it immediately.

Where a native store works it stays the authority and none of this applies. A
native store that is merely unreachable never falls back to the encrypted one,
because credentials split across two backends by session health are worse than
the degraded state `doctor` reports.
