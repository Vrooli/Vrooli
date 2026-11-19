# Experience contracts

This folder is the authored experience contract for React Component Library, using `scenario-experience-spec/v1.1.0`.

## Current contracts

The active pages describe the served Nocturne workbench:

- Home (`/`)
- Components (`/components`, `/catalog` redirects)
- Component (`/assets/:id`)
- Component harness (machine target, not a human page)
- Design studio (`/design`; the legacy design and sketch routes redirect)
- Review (`/review`)
- Coverage (`/coverage`)
- Capabilities (`/capabilities`)
- Activity (`/activity`)
- Settings (`/settings`)
- Create (`/new`)
- Preview (`/assets/:id/preview`)

The `x-redesign` block on each page records the D · Nocturne change. It is extension metadata, not a current-behavior claim.

## Served page map

The information architecture and implementation record are in [../docs/concepts/WORKBENCH-REDESIGN.md](../docs/concepts/WORKBENCH-REDESIGN.md):

| Page | Route | Contract status |
|---|---|---|
| Home | `/` | active; executable page story |
| Components | `/components` | active; executable page story |
| Component | `/assets/:id` | active; executable page story |
| Design | `/design` | active; executable page story |
| Review | `/review` | active; executable page story |
| Coverage | `/coverage` | active; executable page story |
| Capabilities | `/capabilities` | active; executable page story |
| Activity | `/activity` | active; executable page story |
| Settings | `/settings` | active; executable page story |
| Create | `/new` | active; executable page story |
| Preview | `/assets/:id/preview` | active; executable page story |

Every route also has one registered page contract and one executable page story; `node scripts/experience-routes-check.mjs` is the guard.

## Journeys

- `inspect-to-compose` — catalog → component → design studio, preserving version, story and evidence context.
- `review-to-resolution` — review → affected asset → review, preserving the finding context.

Both are `status: draft`: they describe the intended runtime sequence, and the workbench implements the surfaces they cross. They become acceptance evidence only when the runtime journey executor or served journeys prove them.

## Promotion

A page is promoted when:

1. the target route is served and registered in `appRoutes`;
2. the experience route guard resolves exactly one page contract for the route;
3. an executable page story exists;
4. the region binding resolves to a real testid/selector;
5. a real capture supports the declared states;
6. the claim can be promoted to `machine` or attested as `manual`.

Home, Review, Activity and Create met 1–4 and 6 on 2026-11-19. Step 5 is still blocked because the governed capture service returns `503 lifecycle admission closed` on this host; the contracts are active, but the capture-dependent acceptance remains open.

## Rules

- Claims are intent, not descriptions. Do not encode selectors or pixel values in claim text.
- Bindings are volatile; claims are stable.
- Do not add a machine claim without an implementing deterministic check.
- Keep the current page contract honest; use `x-redesign` for planned changes.
- Every served URL needs one page contract and an executable page story.
- Do not mark a page active because a static prototype exists.

## Validation

```bash
cd ui
node scripts/experience-routes-check.mjs
```

Use Experience Manager for parse/reconciliation when a served build exists. A draft page reconciles advisory only; it is not a delivery claim.
