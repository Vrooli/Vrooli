# experience/ — the UX spec

This folder is the scenario's **experience contract**: machine-checkable claims about
what each page must communicate, mirroring what `requirements/` does for business
logic. Both tracks hang off `PRD.md` operational targets.

```
PRD.md            ← shared roof (business OTs + experience OTs)
requirements/     ← what it does           (business track)
experience/       ← what it communicates   (this folder)
bas/              ← evidence producers for both tracks
```

- **Schema**: [`scenario-experience-spec/v1`](../../../.vrooli/schemas/scenario-experience-spec.schema.json) —
  every file declares its `$schema`. Structure: `index.json` (registry) +
  `pages/*.json` (per-page claims) + `journeys/*.json` (cross-page flows).
- **Doctrine** (short version — full rows in
  [`docs/internal/DECISIONS.md`](../docs/internal/DECISIONS.md)):
  - **Claims, not descriptions.** A spec states perceivable outcomes; it never
    inventories a page. Anything unclaimed is free — nothing is flagged for being
    unmentioned. The optional `sketch` block is non-normative (wireframe rendering
    and workshop only).
  - **Tiers.** `machine` claims gate CI (checked against the BAS-captured
    accessibility tree); `manual` claims are human-attested with expiry;
    `aspirational` claims are stated intent with no check yet — advisory, never
    rejected. The language may run ahead of the validator.
  - **Intent vs. bindings.** Claims/elements are stable (WAI-ARIA role + accessible
    name). `bindings` maps element ids to testids/selectors and is the only section
    a restyle or refactor should touch. Bindings are also the selector SSOT that
    `bas/` case scaffolding reads.
  - **Depth is a ladder** (L0 identity → L1 priorities → L2 grounded claims →
    L3 state coverage → L4 journey membership), computed by the parser, never
    stored here.
- **Statuses**: pages here are `draft` — the spec is authored **ahead of the build**
  (design-first), so reconciliation is advisory until a page flips to `active`.

This particular folder is also **OT-P0-005** (self-spec dogfood): experience-manager
is its own first proof. When the validator (OT-P0-001) lands, this spec must
validate green; when the UI is built, these pages flip to `active` one by one and
the machine-tier claims start gating.
