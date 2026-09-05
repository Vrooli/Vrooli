// DOC: docs/internal/COHERENCE-NOTES.md#styling-patterns
import { forwardRef, type ReactNode } from "react";
import {
  Button as LibraryButton,
  type ButtonProps as LibraryButtonProps,
} from "@vrooli/react-component-library/Button/2";

/**
 * Adapter over the library's Button.
 *
 * This file used to be a local `cva` fork. It carried only three variants, no
 * destructive tone, no pending state, and no `white-space: nowrap`, which is
 * why a two-word label like "Re-apply" broke across two lines inside its own
 * button. The file survives as an adapter rather than being deleted because 29
 * modules import `Button` from here: keeping the path stable means a fix lands
 * everywhere at once instead of in 29 separate edits, and the legacy variant
 * vocabulary keeps resolving while call sites migrate.
 *
 * ── Shape ───────────────────────────────────────────────────────────────────
 *
 * The default here is `pill`, which is NOT the library's default (`square`).
 * That override is deliberate and it is the whole reason this file still has a
 * `shape` line in it. The full reasoning lives on the `shape` prop of
 * `@vrooli/react-component-library/Button/2` — read that before changing a
 * call site. The one-line version:
 *
 *     Pill is for controls that act. Square is for controls that repeat.
 *
 * Square is the exception, and it has exactly three cases:
 *
 *   S1  keypad   identical controls tiled in a strip or grid, sized by the
 *                grid rather than by their label — the mobile toolbar keys,
 *                the arrow cluster, the terminal console row.
 *   S2  column   one control repeated once per row down a list or table. A
 *                column of pills reads as a stack of lozenges and the eye
 *                tracks the repeated curve instead of the labels.
 *   S3  welded   the control shares a border with a field — an InputGroup
 *                trailing action, a Send attached to a textarea. It is part of
 *                the field's rectangle, not a button beside one.
 *
 * Everything else is a pill: dialog and form footers, card action rows,
 * full-width CTAs, empty-state CTAs, filter and choice chips, lone icon
 * buttons. One corollary: never mix shapes inside a single action row.
 * Different regions of a panel may differ; one row may not.
 *
 * A previous version of this comment claimed "if a text field is on screen,
 * the buttons are square." That was wrong and is retracted — it keyed on
 * proximity, which is true nearly everywhere, so it collapsed to "square,
 * always." Material 3 ships fully rounded buttons in dialogs; so do we.
 *
 * Two notes for whoever edits a call site:
 *
 *   - `IconButton` (a different library asset) already defaults to `circle`,
 *     which is the same decision as `pill` under a different name. An icon
 *     control that stays on this file's default therefore agrees with the
 *     IconButton next to it, which it did not before.
 *   - `ControlBase` sets `borderRadius` as an INLINE STYLE. A Tailwind
 *     `rounded` / `rounded-md` class in a call site's `className` is inert
 *     against it and will silently do nothing. Pass `shape`, not a class.
 */

/** The vocabulary the fork shipped, mapped onto the library's own. */
const LEGACY_VARIANT = {
  default: "primary",
  outline: "secondary",
  ghost: "ghost",
} as const;

type LegacyVariant = keyof typeof LEGACY_VARIANT;
type LegacySize = "default";

export interface ButtonProps
  extends Omit<LibraryButtonProps, "variant" | "size" | "children"> {
  children: ReactNode;
  variant?: LibraryButtonProps["variant"] | LegacyVariant;
  size?: LibraryButtonProps["size"] | LegacySize;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "default", size = "default", shape = "pill", ...props },
  ref,
) {
  const resolvedVariant =
    variant in LEGACY_VARIANT
      ? LEGACY_VARIANT[variant as LegacyVariant]
      : (variant as LibraryButtonProps["variant"]);
  const resolvedSize = size === "default" ? "md" : (size as LibraryButtonProps["size"]);

  return (
    <LibraryButton
      {...props}
      ref={ref}
      variant={resolvedVariant}
      size={resolvedSize}
      shape={shape}
    />
  );
});

export default Button;
