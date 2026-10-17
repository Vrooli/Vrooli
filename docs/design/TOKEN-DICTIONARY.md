# Design Token Dictionary

This dictionary is generated from `templates/design/_base/tokens.css`. Kit files override values without changing token meaning or tier.

| Property | Tier | Vrooli default | Meaning | Use when |
|---|---|---|---|---|
| `--app-background` | Expression | `var(--color-background)` | Canonical design-system value for app background. | Implementing the named shared component contract. |
| `--app-foreground` | Expression | `var(--color-foreground)` | Canonical design-system value for app foreground. | Implementing the named shared component contract. |
| `--app-muted-foreground` | Expression | `var(--color-muted-foreground)` | Canonical design-system value for app muted foreground. | Implementing the named shared component contract. |
| `--app-shell` | Expression | `var(--color-shell)` | Canonical design-system value for app shell. | Implementing the named shared component contract. |
| `--app-surface` | Expression | `var(--color-surface)` | Canonical design-system value for app surface. | Implementing the named shared component contract. |
| `--app-surface-muted` | Expression | `var(--color-surface-muted)` | Canonical design-system value for app surface muted. | Implementing the named shared component contract. |
| `--app-surface-raised` | Expression | `var(--color-surface-raised)` | Canonical design-system value for app surface raised. | Implementing the named shared component contract. |
| `--badge-border` | Expression | `var(--color-border)` | Canonical design-system value for badge border. | Implementing the named shared component contract. |
| `--border-focus` | Contract | `2px` | Canonical design-system value for border focus. | Implementing the named shared component contract. |
| `--border-hairline` | Rhythm | `1px` | Canonical design-system value for border hairline. | Implementing the named shared component contract. |
| `--border-medium` | Rhythm | `2px` | Canonical design-system value for border medium. | Implementing the named shared component contract. |
| `--border-strong` | Rhythm | `2px` | Canonical design-system value for border strong. | Implementing the named shared component contract. |
| `--border-thin` | Rhythm | `1px` | Canonical design-system value for border thin. | Implementing the named shared component contract. |
| `--color-accent` | Expression | `#0891b2` | Semantic color role for color accent. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-accent-subtle` | Expression | `color-mix(in srgb, var(--color-accent) 14%, transparent)` | Semantic color role for color accent subtle. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-background` | Expression | `#f8fafc` | Semantic color role for color background. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-border` | Expression | `#cbd5e1` | Semantic color role for color border. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-border-strong` | Expression | `color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))` | Semantic color role for color border strong. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-danger` | Expression | `#dc2626` | Semantic color role for color danger. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-danger-border` | Expression | `color-mix(in srgb, var(--color-danger) 38%, var(--color-border))` | Semantic color role for color danger border. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-danger-foreground` | Expression | `color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))` | Semantic color role for color danger foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-danger-foreground-inverse` | Expression | `var(--color-primary-foreground)` | Semantic color role for color danger foreground inverse. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-danger-subtle` | Expression | `color-mix(in srgb, var(--color-danger) 12%, var(--color-surface))` | Semantic color role for color danger subtle. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-field` | Expression | `var(--color-surface)` | Semantic color role for color field. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-focus` | Contract | `#2563eb` | Semantic color role for color focus. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-focus-ring` | Contract | `var(--color-focus)` | Semantic color role for color focus ring. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-foreground` | Expression | `#0f172a` | Semantic color role for color foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-info` | Expression | `#0284c7` | Semantic color role for color info. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-muted-foreground` | Expression | `#64748b` | Semantic color role for color muted foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-on-primary` | Expression | `var(--color-primary-foreground)` | Semantic color role for color on primary. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-overlay` | Expression | `var(--color-shell)` | Semantic color role for color overlay. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-primary` | Expression | `#2563eb` | Semantic color role for color primary. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-primary-foreground` | Expression | `#ffffff` | Semantic color role for color primary foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-primary-hover` | Expression | `color-mix(in srgb, var(--color-primary) 88%, var(--color-foreground))` | Semantic color role for color primary hover. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-primary-strong` | Expression | `var(--color-primary)` | Semantic color role for color primary strong. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-scrim` | Expression | `color-mix(in srgb, var(--color-shell) 52%, transparent)` | Semantic color role for color scrim. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-shell` | Expression | `#020617` | Semantic color role for color shell. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-success` | Expression | `#16a34a` | Semantic color role for color success. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-success-foreground` | Expression | `color-mix(in srgb, var(--color-success) 76%, var(--color-foreground))` | Semantic color role for color success foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-surface` | Expression | `#ffffff` | Semantic color role for color surface. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-surface-muted` | Expression | `#f1f5f9` | Semantic color role for color surface muted. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-surface-raised` | Expression | `#ffffff` | Semantic color role for color surface raised. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-surface-sunken` | Expression | `color-mix(in srgb, var(--color-surface-muted) 72%, var(--color-background))` | Semantic color role for color surface sunken. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-warning` | Expression | `#d97706` | Semantic color role for color warning. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-warning-foreground` | Expression | `color-mix(in srgb, var(--color-warning) 72%, var(--color-foreground))` | Semantic color role for color warning foreground. | Styling the named semantic state or surface without embedding a literal color. |
| `--color-warning-subtle` | Expression | `color-mix(in srgb, var(--color-warning) 16%, var(--color-surface))` | Semantic color role for color warning subtle. | Styling the named semantic state or surface without embedding a literal color. |
| `--content-min-height` | Rhythm | `12rem` | Canonical design-system value for content min height. | Implementing the named shared component contract. |
| `--control-border` | Expression | `1px solid var(--color-border)` | Canonical design-system value for control border. | Implementing the named shared component contract. |
| `--control-height` | Rhythm | `2.75rem` | Canonical design-system value for control height. | Implementing the named shared component contract. |
| `--control-height-lg` | Rhythm | `3.25rem` | Canonical design-system value for control height lg. | Implementing the named shared component contract. |
| `--control-height-sm` | Rhythm | `2.25rem` | Canonical design-system value for control height sm. | Implementing the named shared component contract. |
| `--control-padding` | Rhythm | `var(--space-sm)` | Canonical design-system value for control padding. | Implementing the named shared component contract. |
| `--control-radius` | Expression | `var(--radius-control)` | Canonical design-system value for control radius. | Implementing the named shared component contract. |
| `--control-size-icon` | Rhythm | `40px` | Canonical design-system value for control size icon. | Implementing the named shared component contract. |
| `--control-size-lg` | Rhythm | `44px` | Canonical design-system value for control size lg. | Implementing the named shared component contract. |
| `--control-size-md` | Rhythm | `40px` | Canonical design-system value for control size md. | Implementing the named shared component contract. |
| `--control-size-sm` | Rhythm | `36px` | Canonical design-system value for control size sm. | Implementing the named shared component contract. |
| `--control-size-xl` | Rhythm | `48px` | Canonical design-system value for control size xl. | Implementing the named shared component contract. |
| `--control-size-xs` | Rhythm | `32px` | Canonical design-system value for control size xs. | Implementing the named shared component contract. |
| `--dur-deliberate` | Expression | `400ms` | Motion timing role for dur deliberate. | Animating the named transition with the shared motion language. |
| `--dur-enter` | Expression | `var(--dur-quick)` | Motion timing role for dur enter. | Animating the named transition with the shared motion language. |
| `--dur-fast` | Expression | `var(--dur-instant)` | Motion timing role for dur fast. | Animating the named transition with the shared motion language. |
| `--dur-instant` | Expression | `120ms` | Motion timing role for dur instant. | Animating the named transition with the shared motion language. |
| `--dur-moderate` | Expression | `280ms` | Motion timing role for dur moderate. | Animating the named transition with the shared motion language. |
| `--dur-normal` | Expression | `var(--dur-moderate)` | Motion timing role for dur normal. | Animating the named transition with the shared motion language. |
| `--dur-quick` | Expression | `180ms` | Motion timing role for dur quick. | Animating the named transition with the shared motion language. |
| `--dur-slow` | Expression | `var(--dur-deliberate)` | Motion timing role for dur slow. | Animating the named transition with the shared motion language. |
| `--ease-enter` | Expression | `cubic-bezier(0, 0, 0, 1)` | Motion timing role for ease enter. | Animating the named transition with the shared motion language. |
| `--ease-exit` | Expression | `cubic-bezier(.3, 0, 1, 1)` | Motion timing role for ease exit. | Animating the named transition with the shared motion language. |
| `--ease-standard` | Expression | `cubic-bezier(.2, 0, 0, 1)` | Motion timing role for ease standard. | Animating the named transition with the shared motion language. |
| `--elev-flat` | Expression | `none` | Elevation treatment for elev flat. | Expressing the named depth level without a local shadow literal. |
| `--elev-floating` | Expression | `0 4px 12px rgba(9, 18, 22, 0.12)` | Elevation treatment for elev floating. | Expressing the named depth level without a local shadow literal. |
| `--elev-modal` | Expression | `0 4px 12px rgba(9, 18, 22, .10), 0 16px 48px rgba(9, 18, 22, .18)` | Elevation treatment for elev modal. | Expressing the named depth level without a local shadow literal. |
| `--elev-overlay` | Expression | `0 2px 4px rgba(9, 18, 22, .06), 0 4px 12px rgba(9, 18, 22, .10)` | Elevation treatment for elev overlay. | Expressing the named depth level without a local shadow literal. |
| `--elev-raised` | Expression | `0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)` | Elevation treatment for elev raised. | Expressing the named depth level without a local shadow literal. |
| `--elev-subtle` | Expression | `0 1px 2px rgba(9, 18, 22, .06)` | Elevation treatment for elev subtle. | Expressing the named depth level without a local shadow literal. |
| `--focus-ring` | Contract | `0 0 0 3px color-mix(in srgb, var(--color-focus) 35%, transparent)` | Canonical design-system value for focus ring. | Implementing the named shared component contract. |
| `--focus-ring-color` | Contract | `var(--color-focus)` | Canonical design-system value for focus ring color. | Implementing the named shared component contract. |
| `--focus-ring-offset` | Contract | `2px` | Canonical design-system value for focus ring offset. | Implementing the named shared component contract. |
| `--focus-ring-width` | Contract | `2px` | Canonical design-system value for focus ring width. | Implementing the named shared component contract. |
| `--font-mono` | Expression | `"JetBrains Mono", "Fira Code", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace` | Typography role for font mono. | Applying the named text hierarchy without reconstructing font metrics. |
| `--font-sans` | Expression | `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif` | Typography role for font sans. | Applying the named text hierarchy without reconstructing font metrics. |
| `--font-size-lg` | Expression | `18px` | Typography role for font size lg. | Applying the named text hierarchy without reconstructing font metrics. |
| `--font-size-sm` | Expression | `14px` | Typography role for font size sm. | Applying the named text hierarchy without reconstructing font metrics. |
| `--glow-primary` | Expression | `rgba(51,214,255,.5)` | Ambient glow for glow primary. | Lifting a live or focused figure on dark display surfaces without a local shadow literal. |
| `--icon-size-lg` | Rhythm | `24px` | Canonical design-system value for icon size lg. | Implementing the named shared component contract. |
| `--icon-size-md` | Rhythm | `20px` | Canonical design-system value for icon size md. | Implementing the named shared component contract. |
| `--icon-size-sm` | Rhythm | `16px` | Canonical design-system value for icon size sm. | Implementing the named shared component contract. |
| `--icon-size-xs` | Rhythm | `12px` | Canonical design-system value for icon size xs. | Implementing the named shared component contract. |
| `--layer-alert` | Contract | `700` | Stacking-order contract for layer alert. | Placing the named overlay class in the shared z-order. |
| `--layer-base` | Contract | `0` | Stacking-order contract for layer base. | Placing the named overlay class in the shared z-order. |
| `--layer-dropdown` | Contract | `200` | Stacking-order contract for layer dropdown. | Placing the named overlay class in the shared z-order. |
| `--layer-menu` | Contract | `610` | Stacking-order contract for layer menu. | Placing the named overlay class in the shared z-order. |
| `--layer-modal` | Contract | `400` | Stacking-order contract for layer modal. | Placing the named overlay class in the shared z-order. |
| `--layer-overlay` | Contract | `300` | Stacking-order contract for layer overlay. | Placing the named overlay class in the shared z-order. |
| `--layer-popover` | Contract | `200` | Stacking-order contract for layer popover. | Placing the named overlay class in the shared z-order. |
| `--layer-raised` | Contract | `150` | Stacking-order contract for layer raised. | Placing the named overlay class in the shared z-order. |
| `--layer-sticky` | Contract | `100` | Stacking-order contract for layer sticky. | Placing the named overlay class in the shared z-order. |
| `--layer-toast` | Contract | `500` | Stacking-order contract for layer toast. | Placing the named overlay class in the shared z-order. |
| `--layer-tooltip` | Contract | `600` | Stacking-order contract for layer tooltip. | Placing the named overlay class in the shared z-order. |
| `--motion-fast` | Expression | `var(--dur-quick)` | Motion timing role for motion fast. | Animating the named transition with the shared motion language. |
| `--motion-normal` | Expression | `var(--dur-moderate)` | Motion timing role for motion normal. | Animating the named transition with the shared motion language. |
| `--opacity-disabled` | Contract | `.40` | Canonical design-system value for opacity disabled. | Implementing the named shared component contract. |
| `--opacity-muted` | Expression | `.64` | Canonical design-system value for opacity muted. | Implementing the named shared component contract. |
| `--opacity-scrim` | Expression | `.72` | Canonical design-system value for opacity scrim. | Implementing the named shared component contract. |
| `--overlay-dialog-lg` | Rhythm | `48rem` | Canonical design-system value for overlay dialog lg. | Implementing the named shared component contract. |
| `--overlay-dialog-md` | Rhythm | `36rem` | Canonical design-system value for overlay dialog md. | Implementing the named shared component contract. |
| `--overlay-dialog-sm` | Rhythm | `24rem` | Canonical design-system value for overlay dialog sm. | Implementing the named shared component contract. |
| `--overlay-drawer-top-gap` | Rhythm | `32px` | Canonical design-system value for overlay drawer top gap. | Implementing the named shared component contract. |
| `--overlay-grabber-block` | Rhythm | `4px` | Canonical design-system value for overlay grabber block. | Implementing the named shared component contract. |
| `--overlay-grabber-inline` | Rhythm | `36px` | Canonical design-system value for overlay grabber inline. | Implementing the named shared component contract. |
| `--overlay-menu-align` | Rhythm | `0px` | Canonical design-system value for overlay menu align. | Implementing the named shared component contract. |
| `--panel-padding` | Rhythm | `var(--space-md)` | Canonical design-system value for panel padding. | Implementing the named shared component contract. |
| `--panel-radius` | Expression | `var(--radius-panel)` | Canonical design-system value for panel radius. | Implementing the named shared component contract. |
| `--provenance-absent` | Expression | `var(--color-muted-foreground)` | Provenance encoding for a value that is absent. | Marking where a displayed figure came from so measured, cached, sampled and absent values never share a colour. |
| `--provenance-cached` | Expression | `var(--color-warning)` | Provenance encoding for a value that is cached. | Marking where a displayed figure came from so measured, cached, sampled and absent values never share a colour. |
| `--provenance-measured` | Expression | `var(--color-success)` | Provenance encoding for a value that is measured. | Marking where a displayed figure came from so measured, cached, sampled and absent values never share a colour. |
| `--provenance-sample` | Expression | `#b7a6ff` | Provenance encoding for a value that is sample. | Marking where a displayed figure came from so measured, cached, sampled and absent values never share a colour. |
| `--radius-control` | Expression | `0.375rem` | Corner treatment for radius control. | Rounding the named surface or control role. |
| `--radius-overlay` | Expression | `1rem` | Corner treatment for radius overlay. | Rounding the named surface or control role. |
| `--radius-panel` | Expression | `0.5rem` | Corner treatment for radius panel. | Rounding the named surface or control role. |
| `--radius-pill` | Expression | `9999px` | Corner treatment for radius pill. | Rounding the named surface or control role. |
| `--radius-sheet` | Expression | `1rem` | Corner treatment for radius sheet. | Rounding the named surface or control role. |
| `--scrollbar-thumb` | Expression | `#94a3b8` | Canonical design-system value for scrollbar thumb. | Implementing the named shared component contract. |
| `--scrollbar-thumb-hover` | Expression | `#64748b` | Canonical design-system value for scrollbar thumb hover. | Implementing the named shared component contract. |
| `--sidebar-max-width` | Rhythm | `30rem` | Canonical design-system value for sidebar max width. | Implementing the named shared component contract. |
| `--sidebar-min-width` | Rhythm | `16.25rem` | Canonical design-system value for sidebar min width. | Implementing the named shared component contract. |
| `--sidebar-width` | Rhythm | `20rem` | Canonical design-system value for sidebar width. | Implementing the named shared component contract. |
| `--space-2xl` | Rhythm | `48px` | Spacing-ramp step 2xl. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-2xs` | Rhythm | `8px` | Spacing-ramp step 2xs. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-3xs` | Rhythm | `4px` | Spacing-ramp step 3xs. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-4xl` | Rhythm | `80px` | Spacing-ramp step 4xl. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-4xs` | Rhythm | `4px` | Spacing-ramp step 4xs. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-lg` | Rhythm | `32px` | Spacing-ramp step lg. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-md` | Rhythm | `24px` | Spacing-ramp step md. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-sm` | Rhythm | `16px` | Spacing-ramp step sm. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-xl` | Rhythm | `40px` | Spacing-ramp step xl. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--space-xs` | Rhythm | `12px` | Spacing-ramp step xs. | Choosing internal or inter-component spacing from the shared rhythm. |
| `--spring-expressive` | Expression | `cubic-bezier(.16, 1.2, .3, 1.05)` | Motion timing role for spring expressive. | Animating the named transition with the shared motion language. |
| `--spring-subtle` | Expression | `cubic-bezier(.2, .8, .2, 1.05)` | Motion timing role for spring subtle. | Animating the named transition with the shared motion language. |
| `--tap-target-min` | Contract | `44px` | Canonical design-system value for tap target min. | Implementing the named shared component contract. |
| `--text-body` | Expression | `400 var(--text-body-size) / var(--text-body-line) var(--font-sans)` | Typography role for text body. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-body-line` | Expression | `22px` | Typography role for text body line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-body-size` | Expression | `14px` | Typography role for text body size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-body-sm` | Expression | `400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans)` | Typography role for text body sm. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-body-sm-line` | Expression | `20px` | Typography role for text body sm line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-body-sm-size` | Expression | `13px` | Typography role for text body sm size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-caption` | Expression | `600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)` | Typography role for text caption. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-caption-line` | Expression | `16px` | Typography role for text caption line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-caption-size` | Expression | `11px` | Typography role for text caption size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-code` | Expression | `500 0.8125rem/1.25rem var(--font-mono)` | Typography role for text code. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-display` | Expression | `700 var(--text-display-size) / var(--text-display-line) var(--font-sans)` | Typography role for text display. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-display-line` | Expression | `38px` | Typography role for text display line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-display-size` | Expression | `32px` | Typography role for text display size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-heading` | Expression | `600 var(--text-heading-size) / var(--text-heading-line) var(--font-sans)` | Typography role for text heading. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-heading-lg` | Expression | `600 20px / 26px var(--font-sans)` | Typography role for text heading lg. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-heading-line` | Expression | `24px` | Typography role for text heading line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-heading-size` | Expression | `18px` | Typography role for text heading size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-heading-sm` | Expression | `600 16px / 22px var(--font-sans)` | Typography role for text heading sm. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-label` | Expression | `500 var(--text-label-size) / var(--text-label-line) var(--font-sans)` | Typography role for text label. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-label-line` | Expression | `16px` | Typography role for text label line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-label-size` | Expression | `12px` | Typography role for text label size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-label-tracking` | Expression | `0.005em` | Typography role for text label tracking. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-overline` | Expression | `700 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)` | Typography role for text overline. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-subheading-line` | Expression | `20px` | Typography role for text subheading line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-subheading-size` | Expression | `15px` | Typography role for text subheading size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-subtitle` | Expression | `600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)` | Typography role for text subtitle. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-subtitle-tracking` | Expression | `0` | Typography role for text subtitle tracking. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-title` | Expression | `700 var(--text-title-size) / var(--text-title-line) var(--font-sans)` | Typography role for text title. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-title-line` | Expression | `30px` | Typography role for text title line. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-title-size` | Expression | `24px` | Typography role for text title size. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-title-tracking` | Expression | `-.01em` | Typography role for text title tracking. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-wall` | Expression | `clamp(5rem, 16vw, 20rem)` | Typography role for text wall. | Applying the named text hierarchy without reconstructing font metrics. |
| `--text-xs` | Expression | `12px` | Typography role for text xs. | Applying the named text hierarchy without reconstructing font metrics. |
| `--touch-target` | Contract | `44px` | Canonical design-system value for touch target. | Implementing the named shared component contract. |
| `--tracking-caps` | Expression | `.08em` | Typography role for tracking caps. | Applying the named text hierarchy without reconstructing font metrics. |
| `--tracking-tight` | Expression | `-.02em` | Typography role for tracking tight. | Applying the named text hierarchy without reconstructing font metrics. |
| `--weight-medium` | Expression | `500` | Typography role for weight medium. | Applying the named text hierarchy without reconstructing font metrics. |
