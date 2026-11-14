# Component-library gaps

```shell-ejection
{"archetype":"navigated-console","reason":"AppShell/2 owns the Agent Manager application frame. These files are independent run-detail and dialog surfaces with their own viewport or overlay geometry; treating them as shell chrome would change feature behavior. They remain scoped product surfaces until the corresponding dialog/detail primitives are proven.","files":["ui/src/components/RunDetail.tsx","ui/src/components/patterns/MasterDetail/DetailModal.tsx","ui/src/components/ui/dialog.tsx"]}
```

## Adoption gaps found in W6b (2026-10-29)

Agent Manager now uses RCL `Button/2` (local shadcn button deleted), `StatusBadge/1` for every
run, task and check status chip, `ErrorState/1` (through `components/feedback/ErrorNotice`),
`LoadingState/1`, `AlertDialog/2` for every destructive confirmation, and `EmptyState/1`
(1.8.0 added `density="compact"` for lists, panes and cards). The remaining local primitives
stay for these reasons:

| Local primitive | RCL candidate | Why it is not adopted yet | What would close it |
|---|---|---|---|
| `ui/card.tsx` (12 files) | `Card/1` 1.3.0 | Part styles use `[data-rcl-card] .rcl-card__content` / `__header` selectors (specificity 0,2,0), so a consumer `className` cannot change padding or remove the header rule. Agent Manager cards set their own density (`p-0` lists, `px-4 py-3` headers); swapping would change every card's layout. | Card parts expose a `density`/`inset` prop, or part rules move under `:where()` like `ControlBase`. |
| `ui/badge.tsx` (label pills: model, mode, tag, source) | `Badge/1` primitive, `StatusBadge/1` | `Badge/1` renders `role="status"`, which makes every label a live region; `StatusBadge` always draws a status dot, which is wrong for non-status labels. | A non-live, dot-less label variant (e.g. `Badge` without `role`, or `StatusBadge indicator={false}`). |
| Desktop nav group headings | `AppShell/2` 2.1.2 | `AppShellNavItem` has no group/section field, so the sidebar cannot show the work / insight / system headings that `navigation.tsx` already models (`group`). Adding sections touches the sidebar, rail and phone menu layouts and their story contracts, which is larger than this workstream. | `AppShell` 2.2: optional `group` on items plus `groups` labels, rendered as headings in `sidebar` density and in the phone menu. |
| Formal adoption records | `adoptions link` | Agent Manager imports RCL source by major version and has 0 adoption records. `adoptions link` first requires `@vrooli/ui-selectors` (`scenario-dependency-analyzer deps install npm/@vrooli/ui-selectors --scenario agent-manager --surface ui --version file:../../../packages/ui-selectors --apply`, approved) and then a generated selector registry. | Run that install and link each adopted asset in one pass. |
