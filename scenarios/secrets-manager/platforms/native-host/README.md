# Secrets Manager native host

This directory owns the versioned browser native-messaging boundary. The
protocol library enforces bounded frames, exact enrolled extension identity,
session expiry, one-time nonces, origin/tab/frame/document/item-revision
binding, and revoke/disconnect behavior before an authority adapter can return
a credential.

The checked-in host binary is fail-closed until
`SECRETS_MANAGER_NATIVE_HOST_URL` and
`SECRETS_MANAGER_NATIVE_HOST_TOKEN` configure its HTTP authority adapter. The
adapter also requires the owner token supplied by the extension during
enrollment and unlock. It calls only the narrow Secrets Manager
`/api/v1/native-host/*` contract, and keeps owner/session capabilities in
memory. It writes protocol frames only to stdout and sends diagnostics to
stderr. It does not invoke a shell or treat environment state as an enrollment
authority.

The control plane owns installation. After placing the release host binary at
an absolute path, register the exact released extension identity with:

```bash
vrooli credentials extensions install \
  --host-path /absolute/path/secrets-manager-native-host \
  --extension-id <released-extension-id> \
  --browser chrome
vrooli credentials extensions status \
  --host-path /absolute/path/secrets-manager-native-host \
  --extension-id <released-extension-id> \
  --browser chrome
```

Use `--browser chromium`, `--browser firefox`, or `--browser all` for the
other supported registration targets. Removal is explicit:

```bash
vrooli credentials extensions uninstall \
  --host-path /absolute/path/secrets-manager-native-host \
  --extension-id <released-extension-id> \
  --browser chrome --yes
```

The installer writes only the public native-messaging manifest and, on
Windows, the per-user registry pointer. Chromium uses `allowed_origins`;
Firefox uses `allowed_extensions`. The manifests under `manifests/` remain
reviewable templates for release packaging and troubleshooting. Each OS still
requires a real browser exchange receipt before release support is declared.
