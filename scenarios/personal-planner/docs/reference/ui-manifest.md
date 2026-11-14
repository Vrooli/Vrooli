# UI Manifest Reference

## Shared contract

Read the [UI Manifest Reference shared guide](/scenarios/template-manager/docs/reference/ui-manifest.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Read the scenario's `.vrooli/ui-manifest.json` overlay when present, and
`ui/manifest.json` for the component-path slot map. Record local path
overrides here only when they need rationale beyond the declaration
itself.

**Shell archetype (planned).** Personal Planner uses the
react-component-library **navigated console** archetype — a left rail of
destinations plus a scrolling main pane — configured, never redrawn. The
`ui/manifest.json` shell declaration carries:

- `shell.archetype` = navigated console;
- `shell.asset` — the library shell asset the scenario mounts;
- `shell.entry` — the shell mount point (`ui/src/layout/AppShell.tsx`,
  which mounts the library AppShell with this scenario's nav items and
  router adapter and does not draw chrome);
- `shell.export` — the shell export the scenario adopts.

The five destinations (Today, Plan, Goals, Focus, Review), the
"Planner / Observatory" wordmark, and the Settings/profile entries live in
`ui/src/layout/navItems.tsx` and `BrandMark.tsx`; the Auto/Day/Night
control lives in the shell's `utility` slot. See
[`../concepts/EXPERIENCE.md`](../concepts/EXPERIENCE.md) for the full shell
configuration.

**Observatory theme.** The operator-approved **Observatory** direction
(decision D08) styles the shell: a coordinated day/night landscape that
sits below the working area, expressive editorial serif headings paired
with a readable UI sans and tabular numerals, restrained radii, and no
neon/glass/baked-in text. WCAG 2.2 AA is a visual release gate; time
geometry is truthful (block width is proportional to real duration). If
the navigated-console archetype cannot host the decorative full-bleed
scene band or the proportional timeline, record a scoped `shell-ejection`
in [`component-library-gaps.md`](component-library-gaps.md) naming the
exact `ui/src/` files — pre-1.0 availability alone does not justify a
forced or ejected shell.
