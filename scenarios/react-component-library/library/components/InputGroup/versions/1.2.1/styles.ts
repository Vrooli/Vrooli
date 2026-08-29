/**
 * The whole component is this idea: the field chrome belongs to the wrapper,
 * not to the control inside it. An input that draws its own border has already
 * claimed the shape, so anything placed beside it is necessarily outside it.
 * Move border, background, radius and the focus ring up one level and the
 * adornments, segments and actions all land inside the same outline.
 *
 * Two specificity facts govern the neutralising rules below, and both were
 * measured rather than assumed:
 *
 *  - `[data-rcl-input]` is (0,1,0), so `[data-rcl-input-group] [data-rcl-input]`
 *    at (0,2,0) wins without `!important` and without depending on which sheet
 *    mounted first.
 *  - `[data-rcl-input]:focus-visible` is *also* (0,2,0). A tie resolves by
 *    source order, and `useLibraryStyleSheet` inserts at `head.firstChild`, so
 *    the order depends on component mount order and is not stable. The focus
 *    neutralisers therefore carry a third simple selector to reach (0,3,0).
 *
 * What the neutralisers deliberately do *not* take is `font` and `color`. Both
 * are already declared by `Input` and `Textarea` at (0,1,0), which is exactly
 * weak enough for an ordinary consumer class to win. Restating them here at
 * (0,2,0) would silently beat every adopting scenario's own typography — and
 * the first casualty is the 16px font size a mobile composer sets to stop iOS
 * zooming the viewport on focus. The group owns the box; the host owns the
 * type.
 *
 * Two custom properties are the supported host seam, so a caller can retune
 * the geometry without overriding a rule:
 *
 *   --rcl-group-pad      inline gutter shared by the control and its adornments
 *   --rcl-group-control  minimum block size of the control row
 */
export const inputGroupStyles = `
[data-rcl-input-group] {
  --rcl-group-control: var(--control-size-md, 40px);
  --rcl-group-pad: var(--space-sm, 16px);
  box-sizing: border-box;
  display: flex;
  align-items: stretch;
  inline-size: 100%;
  min-inline-size: 0;
  background: var(--color-field, var(--color-surface));
  color: var(--color-foreground);
  border: var(--border-hairline, 1px) solid var(--color-border);
  border-radius: var(--radius-control, 0.375rem);
  overflow: hidden;
  transition: border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), box-shadow var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1));
}
[data-rcl-input-group][data-size="sm"] { --rcl-group-control: var(--control-size-sm, 36px); --rcl-group-pad: var(--space-xs, 12px); }
[data-rcl-input-group][data-size="lg"] { --rcl-group-control: var(--control-size-lg, 44px); --rcl-group-pad: var(--space-md, 24px); }
/* Half the resting row, deliberately not 9999px.
   A stadium radius is only ever correct on a control that cannot grow. On a
   field that does, 9999px keeps clamping to half the *current* height, so a
   four-line composer bulges into a lozenge. Pinning the radius to half the
   resting row gives both: CSS scales corner radii down proportionally when
   they exceed the box, so at rest this is clamped to a true stadium, while a
   grown field keeps this literal value and reads as a well-rounded rectangle.
   The arithmetic therefore only has to be an upper bound at rest, which is
   why a group with no action slots — resting shorter than this — is still a
   pill. */
[data-rcl-input-group][data-shape="pill"] { border-radius: calc(var(--rcl-group-control) / 2 + var(--space-3xs, 4px) + var(--border-hairline, 1px)); }
[data-rcl-input-group][data-tone="invalid"] { border-color: var(--color-danger); }
[data-rcl-input-group][data-disabled="true"] { opacity: var(--opacity-disabled, .40); pointer-events: none; }

/* One ring for the whole group. Tapping any part of it lights the same
   outline, which is the visible proof that these are one control. */
[data-rcl-input-group]:focus-within { border-color: var(--color-focus); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-focus) 30%, transparent); }
[data-rcl-input-group][data-tone="invalid"]:focus-within { border-color: var(--color-danger); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-danger) 30%, transparent); }

/* The neutralisers. A control inside the group surrenders its own chrome and
   inherits the group's geometry instead. */
[data-rcl-input-group] [data-rcl-input],
[data-rcl-input-group] [data-rcl-textarea],
[data-rcl-input-group] [data-rcl-select],
[data-rcl-input-group] [data-rcl-input-group-control] {
  flex: 1 1 auto;
  inline-size: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  appearance: none;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  outline: none;
  min-block-size: var(--rcl-group-control);
  padding-block: 0;
  padding-inline: var(--rcl-group-pad);
}
[data-rcl-input-group] [data-rcl-textarea] {
  resize: none;
  padding-block: var(--space-2xs, 8px);
  line-height: var(--text-body-line, 22px);
}
/* Interaction states, at (0,4,0) rather than (0,3,0).
   The resting neutraliser above is (0,2,0), which is enough to beat a control's
   own resting rules but *not* its interaction rules: \`Textarea\` repaints its
   surface with \`[data-rcl-textarea]:hover:not(:disabled)\` at (0,3,0), and
   \`Input\` recolours its border at the same weight. Left unhandled, hovering the
   text area paints \`--color-surface-raised\` over the group's own surface, so
   the field goes one colour and the gutter behind its buttons stays another,
   with a hard seam between them. On a touch screen that hover sticks after a
   tap or a scroll, which is how it surfaces in the wild.
   \`:not(:disabled)\` is what carries these to (0,4,0); it costs nothing, since a
   disabled control cannot be hovered into a repaint or focused at all. */
[data-rcl-input-group] [data-rcl-input]:hover:not(:disabled),
[data-rcl-input-group] [data-rcl-textarea]:hover:not(:disabled),
[data-rcl-input-group] [data-rcl-select]:hover:not(:disabled),
[data-rcl-input-group] [data-rcl-input]:focus:not(:disabled),
[data-rcl-input-group] [data-rcl-textarea]:focus:not(:disabled),
[data-rcl-input-group] [data-rcl-select]:focus:not(:disabled),
[data-rcl-input-group] [data-rcl-input]:focus-visible:not(:disabled),
[data-rcl-input-group] [data-rcl-textarea]:focus-visible:not(:disabled),
[data-rcl-input-group] [data-rcl-select]:focus-visible:not(:disabled) {
  background: transparent;
  border-color: transparent;
  box-shadow: none;
  outline: none;
}
[data-rcl-input-group] [data-rcl-input]::placeholder,
[data-rcl-input-group] [data-rcl-textarea]::placeholder,
[data-rcl-input-group] [data-rcl-select]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted, .64); }

/* Field — a positioning context, so a suffix can sit against the value and an
   overlay can register against the text box rather than the whole group. */
[data-rcl-input-group-field] {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 auto;
  min-inline-size: 0;
}

/* Adornment — inside the border, no chrome, never focusable. */
[data-rcl-input-group-adornment] {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  color: var(--color-muted-foreground);
  white-space: nowrap;
  user-select: none;
  pointer-events: none;
}
[data-rcl-input-group-adornment][data-side="leading"] { padding-inline-start: var(--rcl-group-pad); }
[data-rcl-input-group-adornment][data-side="trailing"] { padding-inline-end: var(--rcl-group-pad); }
[data-rcl-input-group-adornment] > svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }
/* A leading adornment supplies that side's gutter, so the control must not
   repeat it. Expressed with the adjacent-sibling combinator rather than
   \`:has()\` so the rule needs no modern-selector support to be correct. */
[data-rcl-input-group-adornment][data-side="leading"] + [data-rcl-input-group-field] [data-rcl-input],
[data-rcl-input-group-adornment][data-side="leading"] + [data-rcl-input-group-field] [data-rcl-textarea],
[data-rcl-input-group-adornment][data-side="leading"] + [data-rcl-input],
[data-rcl-input-group-adornment][data-side="leading"] + [data-rcl-textarea] { padding-inline-start: var(--space-2xs, 8px); }

/* Action — inside the border with chrome of its own, floating in the gutter.
   \`align\` is the prop that keeps this correct on a field that grows: on one
   line \`center\` and \`end\` are identical, and at four lines only \`end\`
   still sits where the eye and the thumb expect it. */
[data-rcl-input-group-action] {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  padding: var(--space-3xs, 4px);
}
[data-rcl-input-group-action][data-align="start"] { align-items: flex-start; }
[data-rcl-input-group-action][data-align="end"] { align-items: flex-end; }

/* Grown — the actions take a row of their own.
   A fixed action lane is right while the field is one line and wrong the moment
   it is four: the lane holds its full width at every height, so a tall field
   pays for a column of empty surface *and* loses the width that would have kept
   it short. Giving the field the whole line and dropping the actions beneath it
   is what chat composers converge on.
   No new markup: the field claims the full line, which pushes the actions onto
   the next flex line, and \`margin-inline-end: auto\` on the start-aligned action
   spreads the row without a wrapper element. \`align\` still governs the resting
   layout — this only changes where the actions live once the field has grown. */
[data-rcl-input-group][data-grown="true"] { flex-wrap: wrap; }
[data-rcl-input-group][data-grown="true"] > [data-rcl-input-group-field],
[data-rcl-input-group][data-grown="true"] > [data-rcl-input],
[data-rcl-input-group][data-grown="true"] > [data-rcl-textarea] { flex: 1 1 100%; }
[data-rcl-input-group][data-grown="true"] > [data-rcl-input-group-action] { align-items: center; }
[data-rcl-input-group][data-grown="true"] > [data-rcl-input-group-action][data-align="start"] { margin-inline-end: auto; }

/* Segment — a full-height peer sharing the outer border. The group clips its
   own corners, so a segment inherits the radius without arithmetic. */
[data-rcl-input-group-segment] {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: var(--space-3xs, 4px);
  min-inline-size: var(--rcl-group-control);
  padding-inline: var(--space-xs, 12px);
  margin: 0;
  appearance: none;
  border: 0;
  background: transparent;
  color: var(--color-foreground);
  font: inherit;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1));
}
[data-rcl-input-group-segment][data-side="leading"] { border-inline-end: var(--border-hairline, 1px) solid var(--color-border); }
[data-rcl-input-group-segment][data-side="trailing"] { border-inline-start: var(--border-hairline, 1px) solid var(--color-border); }
[data-rcl-input-group-segment]:hover:not(:disabled) { background: var(--color-surface-muted); }
[data-rcl-input-group-segment]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled, .40); }
[data-rcl-input-group-segment][data-emphasis="solid"] { background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-input-group-segment][data-emphasis="solid"]:hover:not(:disabled) { background: var(--color-primary); filter: brightness(1.08); }
[data-rcl-input-group-segment] > svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }
/* Inset, because the group is clipped and an outward ring would be cut off.
   The group's own :focus-within ring says "this field"; this says "this part". */
[data-rcl-input-group-segment]:focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus); outline-offset: -3px; }

@media (prefers-reduced-motion: reduce) {
  [data-rcl-input-group], [data-rcl-input-group-segment] { transition: none; }
}
`;
