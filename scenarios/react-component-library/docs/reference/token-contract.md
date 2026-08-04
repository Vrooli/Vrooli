# Design-token contract for adopted assets

Catalog source uses the `app-*` semantic vocabulary. An adopter must either
define these utilities or use the governed adoption translator; an adopted
file must not silently retain a token class its Tailwind config cannot emit.

The current `app-*` contract is:

| Token | Meaning |
| --- | --- |
| `app-background` | application background |
| `app-border` | default border |
| `app-danger` | destructive/error surface |
| `app-foreground` | primary foreground |
| `app-info` | informational accent |
| `app-muted-foreground` | secondary/muted foreground |
| `app-primary` | primary action/accent |
| `app-primary-foreground` | foreground on primary action |
| `app-success` | successful state |
| `app-surface` | raised surface |
| `app-surface-muted` | muted/inset surface |
| `app-warning` | warning state |

`react-component-library` defines the source vocabulary. The mapping is owned
by each adopting scenario in `ui/token-map.json`, next to its isolated
Tailwind configuration. `audio-tools` maps into its `app-*` variables,
`web-console` maps into its `wc-*` variables, and `swarm-manager` maps into its
`slate-*` variables. The library reads the file during apply/reapply; it does
not carry a consumer palette in Go source.

Each mapping entry declares both the generated utility target and the CSS
custom property behind it. A mapping is rejected when an entry is incomplete,
when two roles used by an asset resolve to the same target, or when a declared
contrast pair is below the WCAG AA floor of 4.5:1. The four state roles emitted
by the voice control (`app-info`, `app-primary`, `app-warning`, and
`app-danger`) must remain distinct in every adopting scenario. Because targets
resolve through CSS custom properties, Customize Appearance changes propagate
to adopted controls at runtime.

Translation is recorded in `@vrooliComponentTokenTranslation`, while
`@vrooliComponentSourceSha256` remains the immutable catalog-source digest and
`@vrooliComponentDriftHash` describes the translated copied body.
