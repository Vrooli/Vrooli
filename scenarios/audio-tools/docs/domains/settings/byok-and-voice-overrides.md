# Settings: BYOK, Voice Overrides, Provider Config

This document is the canonical architecture reference for
audio-tools' configuration surface. It explains how BYOK credentials,
voice-id overrides, and provider routing flags are stored, edited,
and pushed into the live provider chains.

Read this first when:

- adding a new persisted operator-tunable lever,
- changing BYOK encryption / fingerprint behavior,
- introducing a new voice-override key shape (`tier:provider-id`),
- debugging "why didn't toggling BYOK off take effect until restart?".

This domain is the only place that writes the encrypted credential
store. Chains (`ttschain`, `sttchain`, `summarizechain`) read
plaintext secrets on demand via the BYOK envelope — they never touch
the store directly.

## Purpose

The `settings` domain owns three persisted singletons:

1. **`provider_config`** — global routing flags (per-tier enable bits,
   resource URLs, availability TTLs).
2. **`byok_credentials`** — encrypted per-(provider, capability) API
   keys with redacted fingerprints for the UI.
3. **`voice_overrides`** — canonical-voice → adapter-voice mappings
   keyed by `tier:provider-id`.

`handlers/settings`
(`api/handlers/settings/connect_handler.go:18`) is the Connect-RPC
surface. The stores live under `internal/store`; the encryptor and
redaction facade live under `internal/byokstore`. The runtime
push-to-chains happens through `chains.Coordinator`.

## Inputs

`SettingsService` exposes seven Connect methods
(`api/handlers/settings/connect_handler.go:37`):

| Method | Inputs | Effect |
|---|---|---|
| `GetProviderConfig` | none | Returns the singleton row, materialising defaults on first read. |
| `UpdateProviderConfig` | optional `byok_enabled`, `vrooli_enabled`, `local_enabled`, `whisper_url`, `kokoro_url`, `ollama_url`, `lpbs_base_url`, `lpbs_app_bundle_key`, `avail_ttl_byok_seconds`, `avail_ttl_vrooli_seconds` (each gated by a `has_*` flag) | Patches the row, then invokes `Coordinator.Reconfigure` to push the new toggles + TTLs into every chain. |
| `ListBYOKCredentials` | none | Returns redacted summaries (`provider_id`, `capability`, `fingerprint`, `created_at`, `last_used_at`). Plaintext is never returned. |
| `UpsertBYOKCredential` | `provider_id`, `capability`, `api_key` | Encrypts and persists; replaces any existing row for the same `(provider, capability)`. |
| `DeleteBYOKCredential` | `provider_id`, `capability` | Removes the row. |
| `GetVoiceOverrides` | none | Returns every `(canonical_voice, tier_provider, adapter_voice)` row. |
| `SetVoiceOverride` | `canonical_voice`, `tier_provider`, `adapter_voice` | Upserts; an empty `adapter_voice` deletes the row. |

Validation:

- `capability` must be `stt`, `tts`, or `summarize`
  (`api/handlers/settings/byok.go:72`); anything else returns
  `CodeInvalidArgument`.
- `provider_id` and `api_key` are required and trimmed
  (`api/handlers/settings/byok.go:38`).
- `canonical_voice` and `tier_provider` are required
  (`api/handlers/settings/voice_overrides.go:31`).

## Outputs

| Method | Wire response |
|---|---|
| `GetProviderConfig` / `UpdateProviderConfig` | `ProviderConfig` with the post-update row plus `updated_at`. |
| `ListBYOKCredentials` | `BYOKCredentialSummary[]` — fingerprint only, never plaintext. |
| `UpsertBYOKCredential` | `BYOKCredentialSummary` — fingerprint of the new key. |
| `DeleteBYOKCredential` | empty. |
| `GetVoiceOverrides` / `SetVoiceOverride` | full `VoiceOverride[]` (Set returns the post-update list so the UI does not have to re-fetch). |

The fingerprint is a stable hash of the key
(`api/internal/byokstore/fingerprint.go`); UIs render it as "ends in
…abcd1234" so operators can confirm which key is loaded without
revealing the secret.

## Internal Chain

### Write paths

```
UpsertBYOKCredential                          UpdateProviderConfig
        │                                          │
        ▼                                          ▼
validateCapability                       build ProviderConfigPatch from has_* flags
        │                                          │
        ▼                                          ▼
byokstore.Store.Upsert                  store.ProviderConfigStore.Update
        │                                          │
        ├─► Encryptor.Seal(key)                    ▼
        │     (AES-GCM 256, key from              chains.Coordinator.Reconfigure
        │      AUDIO_TOOLS_DB_KEY or               │
        │      persisted key file)                 ├─► ttschain.Chain.Reconfigure
        │                                          ├─► sttchain.Chain.Reconfigure
        ▼                                          └─► summarizechain.Chain.Reconfigure
store.BYOKStore.Upsert                            (invalidates availability cache;
        │                                           swaps enable bits + TTLs)
        ▼
SQLite: INSERT … ON CONFLICT DO UPDATE
```

The `chains.Coordinator.Reconfigure` call
(`api/handlers/settings/provider_config.go:71`) is the seam that
makes provider-config edits take effect without a restart. Each
chain's `Reconfigure` (e.g.,
`api/internal/ai/ttschain/chain.go:95`) under-the-hood:

1. Swaps the enable bits and TTLs under its mutex.
2. Zeros the per-tier availability cache so the next call re-probes.

Voice override updates do NOT touch the coordinator — chains read
overrides at request time via `req.VoiceOverrides`, populated upstream
by the request edge (currently from request payload; in future from
the persisted store).

### Read paths

```
Chain dispatch needs a BYOK secret:
        │
        ▼
byokstore.Store.Get(ctx, providerID, capability)
        │
        ▼
store.BYOKStore.Get → BYOKCredential{Cipher, ...}
        │
        ▼
Encryptor.Open(cipher) → plaintext
        │
        ▼
store.BYOKStore.MarkUsed (best-effort, non-blocking)
        │
        ▼
return plaintext to chain
```

The plaintext is only ever held in memory for the duration of a
single chain dispatch. `MarkUsed` updates `last_used_at` so operators
can see which keys are live; failure to update is logged but does not
fail the request (`api/internal/byokstore/store.go:87`).

### Voice override resolution

Voice overrides are read by each TTS provider at synthesis time:

- Local sherpa-onnx/Kokoro consults `overrides["local:kokoro-local"]`
  (`api/internal/ai/ttschain/provider_local.go:73`).
- BYOK adapters consult `overrides["byok:openai-tts"]`,
  `overrides["byok:elevenlabs"]`, etc.
- Vrooli passes the canonical id through unchanged.

The override key shape `tier:provider-id` is the load-bearing
contract: tier names (`local`, `byok`, `vrooli`) plus the same
`provider_id` strings the BYOK registry uses
(`api/internal/byok/registry.go:26`).

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| BYOK encryption | `*byokstore.Encryptor` (`api/internal/byokstore/encryptor.go:19`) | AES-GCM 256, key from `AUDIO_TOOLS_DB_KEY` env or persisted key file | Tests construct an `Encryptor` with a known 32-byte key |
| BYOK persistence | `*store.BYOKStore` (`api/internal/store/byok.go:24`) | SQLite `byok_credentials` table | In-memory SQLite via `setupTestDB` |
| Provider config store | `*store.ProviderConfigStore` (`api/internal/store/provider_config.go:41`) | SQLite singleton row | In-memory SQLite |
| Voice override store | `*store.VoiceOverrideStore` (`api/internal/store/voice_overrides.go:16`) | SQLite `voice_overrides` table | In-memory SQLite |
| Chain reconfigure | `chains.Coordinator.Reconfigure` (`api/internal/ai/chains/coordinator.go`) | Fans out to all three chains | Per-test fake captures the `Config` |

The encryptor seam is the security-critical one. The key is loaded
once at boot from `AUDIO_TOOLS_DB_KEY` if set, otherwise from a
persisted key file. There is no key rotation path today; replacing
the key requires re-upserting every credential.

## Failure Modes

| Cause | Wire response |
|---|---|
| `BYOK store not configured` (`d.BYOK == nil`) | `CodeFailedPrecondition` |
| `voice override store not configured` | `CodeFailedPrecondition` |
| `provider_id` blank, `api_key` blank, `canonical_voice` blank, `tier_provider` blank | `CodeInvalidArgument` |
| Unsupported `capability` value | `CodeInvalidArgument` ("capability must be one of stt\|tts\|summarize") |
| SQLite write error (disk full, locked) | `CodeInternal` |
| Encryptor `Seal` failure | `CodeInternal` (only happens on key misconfiguration at boot) |
| Encryptor `Open` failure (corrupted ciphertext) | Returns `(false, err)` from `byokstore.Store.Get` → chain sees the key as missing and skips the BYOK tier |
| `last_used_at` update fails | Silently swallowed — chain dispatch proceeds with the plaintext key |

Note that the chain layer treats "BYOK key not found" and "BYOK key
decrypt failed" identically: both result in the BYOK tier being
skipped for that request. This is intentional — a partial-state
failure should not surface as a user-visible error during a TTS call;
the operator sees it in the next `ListBYOKCredentials` response where
the row is gone or shows a fresh `created_at`.

## Capacity Notes

The store is a single SQLite file; expected row counts are tiny
(handful of BYOK credentials, single-digit voice overrides, exactly
one provider_config row). Read paths are sub-millisecond; the
encryptor's AES-GCM open is the dominant cost (microseconds).

`byokstore.Store.Get` runs on every chain dispatch that needs a BYOK
key — once per TTS / STT / summarize request when BYOK is enabled.
Decryption is in-process; there is no remote KMS call. If a future
deployment needs HSM-backed keys, the seam to extend is the
`Encryptor` interface.

The `Coordinator.Reconfigure` call is synchronous within
`UpdateProviderConfig`, so the operator's HTTP response only returns
after every chain has applied the new toggles. There is no async
gap during which old toggles are still in effect.

There is no audit log of credential changes today. Adding one is a
known future need; the natural place is a sibling table written from
`byokstore.Store.Upsert` / `Delete`.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — BYOK encryption decision
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — key rotation gap
- [`../../reference/configuration.md`](../../reference/configuration.md) — env vars (`AUDIO_TOOLS_DB_KEY`, resource URLs)
- [`../tts/synthesis-pipeline.md`](../tts/synthesis-pipeline.md) — TTS chain that reads voice overrides
- [`../summarize/chain.md`](../summarize/chain.md) — chain that reads BYOK keys
- `packages/proto/schemas/audio-tools/v1/settings/settings.proto` — wire shape
