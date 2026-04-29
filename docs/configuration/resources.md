# Configuring Resources

Resources are local services scenarios compose with — Postgres, Redis, Vault, Ollama, ComfyUI, and so on. The operator does not pick resources directly; resources are *derived* from the scenarios the operator selected.

## What lives where

| Concern | File | Field |
|---|---|---|
| Whether the operator wants this resource installed | `.vrooli/operator-state.json` | `resources.<name>.enabled` |
| What this resource is and how it runs | `resources/<name>/resource.json` | top-level manifest |
| Resources this one depends on | `resources/<name>/resource.json` | `dependencies` |
| Optional dependencies (graceful degradation) | `resources/<name>/resource.json` | `optional_dependencies` |
| Credential metadata | `resources/<name>/resource.json` | `credentials.env` (per item: bare string OR `secretDescriptor`) |
| Where credential values are stored | `resources/<name>/resource.json` | `credentials.secret_ref` (Vault path) |

## Required vs optional resources

A scenario's manifest declares which resources it needs. Onboarding categorizes resources for the operator into three buckets:

- **Required by selected scenarios** — derived from the union of `dependencies.resources` across enabled scenarios. Onboarding shows these as locked-on once a scenario that needs them is enabled.
- **Optional dependencies** — declared via `optional_dependencies` on a resource. The scenario degrades gracefully if absent. Onboarding presents these as toggleable.
- **Standalone resources** — operator may choose to install for use by future scenarios or direct usage (e.g. running Ollama for ad-hoc local LLM work). Toggleable.

The wizard's resource step has no manual-only path: it always derives from the scenario step. If you want to enable a resource without selecting any scenario that uses it, you fall in the "standalone" bucket — that's the only way the wizard lets you enable a resource the scenarios layer doesn't request.

## Credential descriptors

`credentials.env` historically accepted only bare environment-variable names:

```json
"credentials": {
  "env": ["GEMINI_API_KEY"],
  "secret_ref": "secret/vrooli/gemini"
}
```

This is still valid. For new resources or when enriching existing ones, prefer the `secretDescriptor` form, which surfaces operator-facing metadata directly to the wizard:

```json
"credentials": {
  "env": [
    {
      "env": "GEMINI_API_KEY",
      "label": "Gemini API Key",
      "description": "Google Gemini multimodal LLM. Required for scenarios that call Gemini directly.",
      "classification": "user",
      "required": true,
      "obtain_url": "https://aistudio.google.com/app/apikey",
      "default_hint": "Starts with 'AIza...'"
    }
  ],
  "secret_ref": "secret/vrooli/gemini"
}
```

The two forms can be mixed in the same array — the wizard renders bare strings as a minimal label and rich entries with full metadata.

For the full set of `secretDescriptor` fields and how they map to wizard UI, see [`secrets.md`](secrets.md).

## Adding a new resource

1. Create `resources/<name>/resource.json` conforming to [`resource.schema.json`](../../.vrooli/schemas/resource.schema.json).
2. Declare what scenarios should depend on it (in those scenarios' manifests, not in this one).
3. Use `secretDescriptor` form for any credentials.
4. If the resource needs host tools or safeguards, declare them via `hostTools` / `hostSafeguards` arrays in the resource manifest. The operator opts into those via [`host/`](host/) docs.

The wizard discovers the resource automatically once the manifest is in place.

## See also

- [`scenarios.md`](scenarios.md) — how scenario selection drives resource derivation
- [`secrets.md`](secrets.md) — `secretDescriptor` fields, Vault layout, rotation
- [`host/`](host/) — `hostTools` and `hostSafeguards` resources may declare
