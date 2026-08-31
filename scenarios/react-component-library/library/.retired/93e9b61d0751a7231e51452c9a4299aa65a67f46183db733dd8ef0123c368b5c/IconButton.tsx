/**
 * @libraryId react-component-library:IconButton
 * @displayName IconButton
 * @description Icon-only action with a real hover surface, circular by default, animating whenever its icon changes.
 * @version 3.1.3
 * @tags ["button","icon","accessibility","motion"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.icon-button */
import { cn } from "@vrooli/react-component-library/ClassMerge/1";
import { forwardRef, type ButtonHTMLAttributes, type CSSProperties, type ReactNode } from "react";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { MorphingIcon } from "@vrooli/react-component-library/MorphingIcon/3";
import type { IconMorphMode } from "@vrooli/react-component-library/useIconMorph/1";
import { iconButtonStyles } from "./styles";

/**
 * 3.1.1 — the swap starts before paint.
 *
 * Reported from a frame-by-frame screen recording: pressing the toggle showed
 * the *incoming* icon for a single frame, snapped back to the outgoing one, and
 * only then morphed forward. React commits new children and lets the browser
 * paint before passive effects run, so the frame between commit and effect
 * showed the new icon unhidden. The transition now starts in a layout effect.
 *
 * 3.1.0 — `swapIdentity`, so a control that moves between parents still animates.
 *
 * Found in adoption: web-console's view toggle is rendered from two different
 * parents — floating over the terminal in one mode, inline in the messages
 * toolbar in the other. Switching views therefore unmounts it from one subtree
 * and mounts it in the other, and the fresh instance has no memory of the icon
 * it was showing, so the swap was skipped every time. It looked like the morph
 * was broken; the control was simply new on every toggle.
 *
 * 3.0.2 — an `xs` rung, and pending no longer swallows an icon swap.
 *
 * Two defects found in adoption:
 *
 *   - The scale started at `sm` (36px), but `--control-size-xs` (32px) exists
 *     and real toolbars are built on it. A 32px strip had no rung to adopt, so
 *     every converted control in it grew and stopped matching its neighbours.
 *   - `pending` hid the whole glyph, and a caller whose icon changes *as* the
 *     pending window opens — which is the normal shape of "switch view, then
 *     load it" — played its entire swap animation behind `visibility: hidden`.
 *     The spinner now overlays the glyph instead of replacing it, so a swap
 *     that coincides with a pending window is still seen.
 *
 * 3.0.1 — takes MorphingIcon 3.0.1, whose stylesheet reaches consumers.
 *
 * 3.0.0 paired with MorphingIcon 3.0.0, which shipped its layering rules as a
 * side-effect `.css` import. That import is dropped when the primitive is
 * reached transitively through a package subpath, which is exactly how this
 * component reaches it — so the morph frame stacked below the live icon
 * instead of overlaying it, and the live icon was never hidden. The defect was
 * invisible in MorphingIcon's own stories, where it is the bundle entry.
 *
 * 3.0.0 — an icon button rather than a square text button.
 *
 * 2.x forwarded to `Pressable`/`ControlBase` with `shape="square"` hard-coded
 * and `variant="ghost"` as the default. Those two choices interact badly:
 * `ControlBase`'s hover is `filter: brightness()` plus a lift and a shadow,
 * written for a filled surface, so a transparent icon button hovered with no
 * visible surface change and a drop shadow cast by nothing. The evidence that
 * this was wrong is that every adopter overrode it — both call sites in
 * web-console passed `variant="secondary"` and re-declared the tap target by
 * hand.
 *
 * What changed:
 *   - Circular by default, with the shape exposed. 2.x could not express a
 *     circle at all; `--radius-control` is 6px.
 *   - `ghost` paints a real tinted surface on hover and a deeper one on press,
 *     and nothing moves. `soft` is the standing-surface treatment one prop away.
 *   - `selected` sets `aria-pressed` and its own surface, replacing hand-rolled
 *     `data-active` plus inline styles at the call site.
 *   - The comfortable tap target is honoured on coarse pointers, so call sites
 *     stop patching `min-h-11 min-w-11`.
 *   - The icon animates whenever it changes, with no call-site work.
 *
 * Migration from 2.x: `variant` still accepts the old names. `secondary` and
 * `outline` map to `soft`, `primary`/`default` to `solid`, `danger` and its
 * synonyms to `danger`, and everything else to `ghost`.
 */

export type IconButtonSurface = "ghost" | "soft" | "solid" | "danger";
export type IconButtonShape = "circle" | "rounded" | "square";
export type IconButtonSize = "xs" | "sm" | "md" | "lg";

/** 2.x variant names, kept so existing call sites compile unchanged. */
export type LegacyIconButtonVariant =
  | "primary"
  | "secondary"
  | "ghost"
  | "danger"
  | "default"
  | "outline"
  | "destructive"
  | "info"
  | "success"
  | "warning"
  | "error"
  | "pipeline";

const SURFACE_BY_LEGACY: Record<LegacyIconButtonVariant, IconButtonSurface> = {
  ghost: "ghost",
  secondary: "soft",
  outline: "soft",
  info: "soft",
  success: "soft",
  warning: "soft",
  pipeline: "soft",
  primary: "solid",
  default: "solid",
  danger: "danger",
  destructive: "danger",
  error: "danger",
};

const CONTROL_SIZE: Record<IconButtonSize, string> = {
  // 32px. Dense toolbars are built on this rung; without it every control
  // adopted into one grows and breaks the row it joined.
  xs: "var(--control-size-xs, 32px)",
  sm: "var(--control-size-sm, 36px)",
  md: "var(--control-size-md, 40px)",
  lg: "var(--control-size-lg, 44px)",
};

/**
 * The glyph rung handed to MorphingIcon. `xs` and `sm` share the 16px glyph;
 * only the control box differs between them.
 */
const MORPH_SIZE: Record<IconButtonSize, "sm" | "md" | "lg"> = {
  xs: "sm",
  sm: "sm",
  md: "md",
  lg: "lg",
};

const GLYPH_SIZE: Record<IconButtonSize, string> = {
  xs: "var(--icon-size-sm, 16px)",
  sm: "var(--icon-size-sm, 16px)",
  md: "var(--icon-size-md, 20px)",
  lg: "var(--icon-size-lg, 24px)",
};

export interface IconButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color" | "children"> {
  /** Required: an icon-only control has no other accessible name. */
  "aria-label": string;
  /** The icon. Swapping it animates; see `morph`. */
  children: ReactNode;
  surface?: IconButtonSurface;
  /** @deprecated Use `surface`. Mapped onto the four real surfaces. */
  variant?: LegacyIconButtonVariant;
  shape?: IconButtonShape;
  size?: IconButtonSize;
  /**
   * Toggle state. Sets `aria-pressed`, which is what actually communicates a
   * toggle to assistive technology — a coloured background alone does not.
   */
  selected?: boolean;
  pending?: boolean;
  pendingLabel?: string;
  /**
   * How an icon change animates. `auto` (default) animates every swap and
   * upgrades to a path morph only when the two glyphs measure compatible;
   * `none` swaps instantly.
   */
  morph?: IconMorphMode;
  /**
   * A stable identity for this control, so an icon change still animates when
   * the control remounts. Needed when a layout renders the same logical control
   * from different parents per mode; without it React discards the instance and
   * the swap is silently skipped. Must be unique among controls.
   */
  swapIdentity?: string;
  /**
   * Identity of the current icon. Defaults to the child's component identity,
   * which separates icons from any library without call-site work. Supply it
   * when one component renders different art from its props.
   */
  iconKey?: string;
  /** Suppress the native tooltip that otherwise mirrors `aria-label`. */
  disableTooltip?: boolean;
  /**
   * Opt out of the coarse-pointer tap-target floor. Only for controls packed
   * into a deliberately dense strip where the caller owns hit testing.
   */
  denseTapTarget?: boolean;
}

/**
 * `withClassName` is deliberately not used here.
 *
 * That helper wraps the component in a plain function component, which cannot
 * receive a ref — a `ref` passed by a caller lands on the wrapper and is
 * silently dropped. An icon button is exactly the control callers need a handle
 * on: to focus it after a dialog closes, to anchor a popover, to measure it.
 * `Pressable` and `ControlBase` set the precedent of forwarding directly, and
 * this component owns a single host element, so the class seam is just `cn()`
 * on the root.
 */
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  {
    "aria-label": ariaLabel,
    children,
    surface,
    // The deprecation shim itself: this is the one place that must still
    // read `variant`, precisely so 2.x call sites keep working.
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    variant,
    shape = "circle",
    size = "md",
    selected,
    pending = false,
    pendingLabel = "Working…",
    morph = "auto",
    iconKey,
    swapIdentity,
    disableTooltip = false,
    denseTapTarget = false,
    className,
    disabled,
    style,
    title,
    type = "button",
    "aria-pressed": ariaPressed,
    "aria-busy": ariaBusy,
    ...props
  },
  ref,
) {
  useLibraryStyleSheet("icon-button", iconButtonStyles);

  const resolvedSurface: IconButtonSurface =
    surface ?? (variant ? SURFACE_BY_LEGACY[variant] : "ghost");

  const box = CONTROL_SIZE[size];
  const controlStyle: CSSProperties = {
    inlineSize: box,
    blockSize: box,
    ...style,
  };

  return (
    <button
      data-testid="controls.icon-button"
      {...props}
      ref={ref}
      type={type}
      className={cn(className)}
      style={controlStyle}
      aria-label={ariaLabel}
      title={disableTooltip ? undefined : (title ?? ariaLabel)}
      disabled={disabled || pending}
      aria-busy={ariaBusy ?? (pending || undefined)}
      // `selected` is a toggle; leaving it undefined keeps the button a plain
      // action rather than announcing an unpressed toggle that does not exist.
      aria-pressed={ariaPressed ?? (selected === undefined ? undefined : selected)}
      data-rcl-icon-button=""
      data-rcl-surface={resolvedSurface}
      data-rcl-shape={shape}
      data-rcl-size={size}
      data-rcl-pending={pending ? "true" : "false"}
      data-rcl-tap-target={denseTapTarget ? "dense" : "comfortable"}
    >
      <span
        data-rcl-icon-button-glyph=""
        style={{ inlineSize: GLYPH_SIZE[size], blockSize: GLYPH_SIZE[size] }}
      >
        <MorphingIcon
          morph={morph}
          iconKey={iconKey}
          swapIdentity={swapIdentity}
          size={MORPH_SIZE[size]}
        >
          {children}
        </MorphingIcon>
      </span>
      {pending ? (
        <>
          <span aria-hidden="true" data-rcl-icon-button-spinner="" />
          <span className="rcl-visually-hidden" aria-live="polite">
            {pendingLabel}
          </span>
        </>
      ) : null}
    </button>
  );
});
