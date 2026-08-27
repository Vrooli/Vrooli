# Experience contract authoring

Every catalog asset declares one semantic `kind` in its manifest. The asset
scaffold uses that kind to create an experience contract with a starting claim
set. The claim set is a reviewable default, not permission to keep a claim that
does not describe the asset.

## Kind presets

| Kind | Claims scaffolded by default |
| --- | --- |
| `control` | `tap-target-size`, `content-not-clipped`, `state-contrast`, `padding`, `keyboard-reachable`, `size-parity` |
| `input` | `accessible-name`, `error-association`, `state-contrast`, `keyboard-reachable`, `font-size` |
| `surface` | `no-document-horizontal-overflow`, `spacing`, `heading-hierarchy` |
| `overlay` | `focus-contained`, `layered-dismissal`, `focus-restored`, `reading-order` |
| `shell` | `chrome-pinned`, `viewport-fill`, `safe-area-tap-targets`, `no-document-horizontal-overflow` |

The presets contain only evaluator names implemented by Experience Manager.
The scaffold test fails if a preset names an evaluator that is not registered.
Authors may add or remove claims when the asset anatomy requires it, but the
result must remain truthful and machine-verifiable where a machine evaluator
exists.

## Vacuous-contract ratchet

A contract with no substantive machine claim is vacuous. A new or changed asset
with a vacuous contract is an error and cannot pass catalog validation. An
unchanged historical asset may remain in
`library/vacuous-allowlist.json` only when the entry has a written reason.

The allowlist is a shrinking migration record:

- a new entry is rejected;
- a duplicate, unknown path, or missing reason is rejected;
- changing an allowlisted source requires removing its entry and authoring a
  real contract;
- existing entries may be removed as contracts become substantive.

Use a real machine claim whenever the evaluator can observe the behaviour. Use
`manual-review` for a requirement that still needs human judgement, and state
the reason in the contract. Do not use a populated list of unsupported claim
names to make a contract appear non-vacuous.
