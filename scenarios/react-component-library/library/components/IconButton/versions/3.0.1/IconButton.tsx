/**
 * @libraryId react-component-library:IconButton
 * @displayName IconButton
 * @description Icon-only action with a real hover surface, circular by default, animating whenever its icon changes.
 * @version 3.0.1
 * @tags ["button","icon","accessibility","motion"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.icon-button */
import { cn } from "@vrooli/react-component-library/ClassMerge/1.0.1";
import { forwardRef, type ButtonHTMLAttributes, type CSSProperties, type ReactNode } from "react";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { MorphingIcon } from "@vrooli/react-component-library/MorphingIcon/3.0.1";
import type { IconMorphMode } from "@vrooli/react-component-library/useIconMorph/1.0.0";
import { iconButtonStyles } from "./styles";

/**
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
export type IconButtonSize = "sm" | "md" | "lg";

/** 2.x variant names, kept so existing call sites compile unchanged. */
export type LegacyIconButtonVariant =
  | "primary" | "secondary" | "ghost" | "danger" | "default" | "outline"
  | "destructive" | "info" | "success" | "warning" | "error" | "pipeline";

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
  sm: "var(--control-size-sm, 36px)",
  md: "var(--control-size-md, 40px)",
  lg: "var(--control-size-lg, 44px)",
};

const GLYPH_SIZE: Record<IconButtonSize, string> = {
  sm: "var(--icon-size-sm, 1rem)",
  md: "var(--icon-size-md, 1.25rem)",
  lg: "var(--icon-size-lg, 1.5rem)",
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
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function IconButton(
    {
      "aria-label": ariaLabel,
      children,
      surface,
      variant,
      shape = "circle",
      size = "md",
      selected,
      pending = false,
      pendingLabel = "Working…",
      morph = "auto",
      iconKey,
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
        <span data-rcl-icon-button-glyph="" style={{ inlineSize: GLYPH_SIZE[size], blockSize: GLYPH_SIZE[size] }}>
          <MorphingIcon morph={morph} iconKey={iconKey} size={size}>
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
  },
);
