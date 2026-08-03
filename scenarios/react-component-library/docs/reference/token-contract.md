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

`react-component-library` defines the source vocabulary. `audio-tools` also
defines `app-*`. `web-console` defines `wc-*`; adoption translates the catalog
tokens to its `wc` semantic equivalents. `swarm-manager` defines its palette
under `slate-*`; adoption translates to that namespace. New consumers must
declare a supported namespace in their isolated Tailwind configuration before
adoption is allowed.

Translation is recorded in `@vrooliComponentTokenTranslation`, while
`@vrooliComponentSourceSha256` remains the immutable catalog-source digest and
`@vrooliComponentDriftHash` describes the translated copied body.
