# Product line: the star-named line

> Operator-attributed. The operator named this line on 2026-09-15.

## The line

| Order | Product | Product name origin | Scenario | Status |
|---|---|---|---|---|
| 1 | Aquila | The eagle constellation | web-console | Launching 2026-09-17 |
| 2 | Vega | The brightest star in Lyra | system-monitor | Launched 2026-09-17 |
| 3 | Rigel | The blue supergiant in Orion | not yet mapped | Planned |
| 4 | Mizar | The double star in the Big Dipper | not yet mapped | Planned |
| 5 | Pan | A moon of Saturn | not yet mapped | Planned |
| 6 | Messier | The deep-sky catalogue | not yet mapped | Planned |

Aquila and Vega are mapped to scenarios today. Mapping the remaining names to
scenarios is operator work and is out of scope for the brand-identity pipeline.

## Resolving a scenario to its brand slug and landing route

This table is narrative. Two machine-readable sources are authoritative.

**Brand slug** — the scenario declares it in its own manifest, and the control
plane reports that declaration:

```bash
vrooli scenario info web-console --json | jq -r '.scenario.branding.brand'   # aquila
jq -r '.branding.brand' scenarios/web-console/.vrooli/service.json           # same value
```

`branding.brand` is schema-constrained in `.vrooli/schemas/service.schema.json`
and is the value `brand-manager apply` matches a brand's `slug` against. Do not
infer the slug from `service.displayName`; a scenario with no `branding` block
declares no brand.

**Landing route** — owned by landing-page-business-suite, which also names the
apps that have no brand-manager brand. The variant is an argument; `control` is
the serving fallback and the metrics baseline (`api/internal/metrics/service.go`,
`.vrooli/fallback/fallback.json`), which is what an unassigned visitor sees — the
repository declares no variant as authoritative for a launch.

```bash
cd scenarios/landing-page-business-suite/config/presentations/<variant>
jq -r '.apps[] | "\(.key) -> /apps/\(.slug) (enabled=\(.enabled), \(.visibility)/\(.publication))"' \
  "revisions/$(jq -r .active_revision head.json).json"
```

`apps` is top-level in the revision document. `bundle` holds only `key`, `name`,
`app_order`, `page_id`, `empty_page_id` and locales — there is no `bundle.apps`.

Being listed is not the same as being reachable. `eligibleApps`
(`api/internal/presentation/resolve.go`) requires `enabled && visibility ==
public && publication == published`. Today only Aquila and Vega pass; the other
entries are disabled private drafts.

The two sources agree wherever both are set; for an app with no `branding.brand`
the landing slug falls back to the scenario key.

Two edge cases the command does not handle: a variant with no `active_revision`
resolves `revisions/null.json` and fails with file-not-found (the API returns
`ErrPresentationNotFound`), and a bundle with no eligible apps renders
`bundle.empty_page_id` — a live page with zero `/apps/*` routes.

Verified across all 7 variants on 2026-09-17: every one resolves to the same
content-addressed revision, so the mapping is variant-independent today. The
variant argument still matters, because nothing enforces that agreement.

## Shared identity

Every product in the line shares:

- **Container style `constellation-midnight`** — one rounded-square midnight tile
  with a cyan accent glow. Defined in
  `scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`.
- **Motif rule** — each product uses its constellation or catalog figure as line
  art with glowing stars on a transparent background. The mark differs per
  product; only the mark and the accent colour change.

This keeps the line visually coherent while each product keeps a distinct mark.

For imagery *about* one of these products — landing hero, OG or social card,
store or launch imagery, in-app illustration — this line's identity is the
authority. The Vrooli project image style in
[`IMAGE_STYLE.md`](IMAGE_STYLE.md) governs imagery that speaks for Vrooli as a
whole and does not apply here; see that file's "Scope and precedence".

## Pipeline

The brand-identity pipeline turns a concept into an applied, validated icon set
in five commands, so each later launch needs only a brand, candidates, a pick and
one `apply run`. See
`scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`.
