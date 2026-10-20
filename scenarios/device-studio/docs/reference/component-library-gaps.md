# Component library adoption and gaps

## Purpose Of This Document

Separate shared components from product-specific controls. The scoped top-navigation adaptation below is scenario-owned; it does not authorize editing a shared release or creating a new component release.

## Ejections

### Device Studio top navigation

The approved Light-table boards use one horizontal navigation bar above the workspace. The inspected `AppShell/2` configuration supports a sidebar or rail on desktop and tabs or a drawer on mobile; its `header` slot does not remove those navigation surfaces. `AppNavigation/1` also resolves only to sidebar, drawer, or bottom navigation at the inspected version. The owner source contains an `eject` handler, but the current adoption command registration and `cli/manifest.json` omit it, so no `adoptions eject` CLI command is available.

Device Studio therefore owns the small semantic top-navigation shell in `ui/src/layout/AppShell.tsx`. It preserves the skip link, router links, current-page state, focus styling, theme and locale providers. The shell does not replace any shared release files; forms, buttons and dialogs continue to use RCL components. Revisit this local shell when RCL offers a supported top-navigation/no-sidebar mode or an owner-managed ejection workflow.

## Missing Assets

| Need | Current finding | Intended action |
|---|---|---|
| Complete account/connection surface | Catalogued AccountSettingsSurface returns null in inspected version | Draft/qualify at RCL owner using working MonetizationAccount/AuthClient pieces |
| Physical shot stage and screen calibration | Existing Web Console inspector strips animation/replaces lights | Build Device Studio stage against shared3D primitives; assess reusable controls |
| Motion/keyframe inspector | Product-specific semantics not established by generic components | Adopt shared input/timeline affordances where suitable; record reusable gap |
| Reference comparison/review | Needs exact revision, dimensions, held-out views and scoped verdict | Compose shared comparison/layout controls; domain decisions remain local |

These are planned adoption decisions, not proof that source UI implements them.

## Cross-References

[DESIGN.md](../../DESIGN.md), [choosing UI](../guides/choosing-ui.md), [research](../research/README.md).

## Scoped shell ejection for the approved Light-table

```shell-ejection
{"archetype":"navigated-console","reason":"The approved Device Studio Light-table boards (docs/mockups/revision-2/projects-paired.png and shot-paired.png) require a horizontal top-navigation header. The adopted AppShell/2 only supports sidebar or rail on desktop and tabs or drawer on mobile; its header slot cannot remove those navigation surfaces. Keep the shell exception scoped to the scenario AppShell entry; use shared library controls within product pages.","files":["ui/src/layout/AppShell.tsx"]}
```

This is the ui-health file-scoped ownership exception for the current shell entry. It does not change the declared `navigated-console` / `AppShell` adoption and does not exempt other product files from shared-library ownership checks. Revisit when the library supports the approved top-navigation composition.
