/**
 * @libraryId react-component-library:Button
 * @displayName Button
 * @description The primary action control: a labelled, token-bound button that acknowledges activation immediately, expresses intent through semantic variants, and keeps stable dimensions while it moves through pending, success, and failure.
 * @version 2.2.6
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.button */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1";
import { Pressable } from "@vrooli/react-component-library/Pressable/1";
import { Icon, type IconName, type IconSize } from "@vrooli/react-component-library/Icon/1";
export const BUTTON_VARIANTS = ["primary", "secondary", "ghost", "danger"] as const;
export const BUTTON_SIZES = ["xs", "sm", "md", "lg", "xl", "icon"] as const;
export const BUTTON_DENSITIES = ["comfortable", "compact"] as const;
/**
 * The two shapes a button may take.
 *
 * `pill` is `--radius-pill` (a full stadium); `square` is `--radius-control`
 * (a 6px rounded rectangle — "square" names the corner family, not a 0 radius).
 *
 * `IconButton` expresses the same decision with a different vocabulary:
 * its `circle` is this `pill`, and its `rounded` is this `square`. They are the
 * same two shapes under three names, which is a known wart. When a Button and
 * an IconButton sit in the same row, pair `pill`/`circle` or `square`/`rounded`
 * — never one of each.
 */
export const BUTTON_SHAPES = ["square", "pill"] as const;

export type ButtonVariant = ControlVariant;
export type ButtonSize = ControlSize;

export interface ButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  density?: ControlDensity;
  /**
   * ── Choosing a shape ──────────────────────────────────────────────────────
   *
   *     Pill is for controls that ACT. Square is for controls that REPEAT.
   *
   * Square is the exception, and it has exactly three cases:
   *
   *   S1  keypad   Identical controls tiled in a strip or grid, sized by the
   *                grid rather than by their own label — a key row, an arrow
   *                cluster, a segmented track drawn as adjacent keys. Curves
   *                fight a grid, and pills spend horizontal budget the keys
   *                need.
   *
   *   S2  column   One control repeated once per row down a list or table —
   *                a `Fix` per drift row, a `Delete` per profile. A column of
   *                pills reads as a stack of lozenges and the eye starts
   *                tracking the repeated curve instead of the labels.
   *
   *   S3  welded   The control shares a border with a field: an `InputGroup`
   *                trailing action, a Send attached to a `Textarea`. It is
   *                part of the field's rectangle, not a button beside one.
   *
   * Everything else is a pill: dialog and form footers, card action rows,
   * full-width CTAs, empty-state CTAs, filter and choice chips, and lone icon
   * buttons. One corollary — never mix shapes inside a single action row.
   * Different regions of a panel may differ; one row may not.
   *
   * Two rules that are NOT the doctrine, and have been tried:
   *
   *   - "Square when other geometry is nearby." Proximity is true almost
   *     everywhere, so this collapses to "square, always." Repetition is the
   *     axis that actually discriminates.
   *   - "Square whenever a text field is on screen." Wrong. Material 3 ships
   *     fully rounded buttons inside dialogs and forms; only S3 (welded to the
   *     field) is about fields at all.
   *
   * The default is `square` because it is the shape that never looks actively
   * broken in an un-designed surface, not because it is the usual answer — by
   * the doctrine above, most buttons in a product surface are pills. A
   * consumer that has made the call once may flip the default in its own
   * adapter rather than annotating every call site; `web-console`'s
   * `components/ui/button.tsx` defaults to `pill` and pins `shape="square"` on
   * its ~39 S1/S2/S3 sites.
   *
   * NOTE: `ControlBase` applies `borderRadius` as an INLINE STYLE. A Tailwind
   * `rounded` / `rounded-md` class in a caller's `className` is inert against
   * it and will silently do nothing. Pass this prop, not a class.
   *
   * @default "square"
   */
  shape?: ControlShape;
  /** Preferred path for icons from the governed library registry. */
  iconName?: IconName;
  /** Compatibility path for custom icon content not present in the registry. */
  icon?: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { children, icon, iconName, pending, pendingLabel, size = "md", variant = "primary", ...props },
  ref,
) {
  const testId = (
    props as ButtonHTMLAttributes<HTMLButtonElement> & {
      "data-testid"?: string;
    }
  )["data-testid"];
  return (
    <Pressable
      {...props}
      ref={ref}
      data-testid={testId ?? "controls.button"}
      tone={variant}
      size={size}
      pending={pending}
      pendingLabel={pendingLabel}
    >
      {iconName ? (
        <Icon
          name={iconName}
          size={controlIconSize(size)}
          data-testid="button-icon"
          data-control-slot="icon"
          style={{ color: "currentColor", flex: "0 0 auto" }}
        />
      ) : icon ? (
        <span
          data-testid="button-icon"
          data-control-slot="icon"
          aria-hidden="true"
          style={{ flex: "0 0 auto" }}
        >
          {icon}
        </span>
      ) : null}
      <span
        data-testid="button-label"
        data-control-slot="label"
        style={{
          // inline-flex, not the default block, so that an icon passed inside
          // `children` still sits on the label's line. Consumers on the
          // compatibility path write `<Button><Plus className="me-1.5" />Add
          // machine</Button>`, and a CSS reset that sets `svg { display:
          // block }` — Tailwind's preflight does — then made that svg a
          // block-level box inside this span, breaking the line and stacking
          // the glyph above its own label. `white-space: nowrap` cannot
          // prevent that, because the break is a box-level break, not a text
          // wrap. This mirrors what `[data-rcl-pressable-label]` one level up
          // already does, so the ellipsis behaviour below is unchanged.
          display: "inline-flex",
          alignItems: "center",
          minWidth: 0,
          maxWidth: "100%",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {children}
      </span>
    </Pressable>
  );
});

function controlIconSize(size: ControlSize): IconSize {
  if (size === "xs" || size === "sm") return "sm";
  if (size === "lg" || size === "xl") return "lg";
  return "md";
}
