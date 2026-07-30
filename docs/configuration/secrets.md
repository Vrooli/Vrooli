# Configuring credentials

Vrooli declares credentials only in resource and scenario manifests. A
descriptor identifies a credential without selecting a storage backend:

```json
"credentials": {"descriptors": [{
  "logical_id": "vrooli/openrouter",
  "field": "api-key",
  "env": "OPENROUTER_API_KEY",
  "label": "OpenRouter API Key",
  "required": true,
  "obtain_url": "https://openrouter.ai/keys"
}]}
```

`logical_id` and `field` are stable backend-neutral names. `env` is only the
process-scoped injection name; it is not durable storage. The local and desktop
authority is a probed native OS secure store. Vault may be a scoped mirror or a
capability-specific service, but it is not the ordinary credential authority.

Provision through onboarding or standard input, never a command argument:

```bash
printf '%s' "$OPERATOR_VALUE" | vrooli credentials provision \
  --identity vrooli/openrouter --field api-key
```

Use `vrooli credentials status` for metadata-safe presence checks. Runtime
code resolves credentials through the control-plane authority into the target
process only. It must not read YAML inventories, resource-private credential
files, generic environment fallbacks, or Vault content directly.

Recovery export and restore are explicit encrypted operations protected by an
operator-held passphrase. Unsupported host stores fail closed; there is no
plaintext fallback. Retired `config/secrets.yaml` files are migration inputs
only and never a runtime contract.

Create a recovery bundle by naming the credential metadata and passing the
passphrase through standard input. The output path is created once with owner-
only permissions; neither command prints a value or passphrase:

```bash
printf '%s' "$RECOVERY_PASSPHRASE" | vrooli credentials recovery export \
  --entry vrooli/openrouter:api-key --output ./vrooli-recovery.bundle

printf '%s' "$RECOVERY_PASSPHRASE" | vrooli credentials recovery restore \
  --input ./vrooli-recovery.bundle
```
