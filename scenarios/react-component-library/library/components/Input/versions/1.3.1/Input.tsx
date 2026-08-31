/**
 * @libraryId react-component-library:Input
 * @displayName Input
 * @description Token-bound text input with mobile-safe font sizing and accessible focus styling.
 * @version 1.3.1
 * @tags ["form","interactive"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { forwardRef, type InputHTMLAttributes } from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export const INPUT_MODES = ["controlled", "uncontrolled"] as const;
export const INPUT_SIZES = ["sm", "md", "lg"] as const;
export const INPUT_TONES = ["default", "invalid"] as const;
/**
 * The parts an input can be composed with. Declared here since 1.1.x and
 * unimplemented until `InputGroup` — the group is what actually renders a
 * prefix and suffix inside this control's border, because chrome has to move
 * off the input before anything can join it.
 *
 * @see react-component-library:InputGroup
 */
export const INPUT_PARTS = ["prefix", "control", "suffix"] as const;

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  "data-testid"?: string;
};

const styleSheet = `
[data-rcl-input] {
  box-sizing: border-box;
  width: 100%;
  min-height: var(--tap-target-min);
  border: var(--border-hairline) solid var(--color-border);
  border-radius: var(--radius-control);
  background: var(--color-field, var(--color-surface));
  color: var(--color-foreground);
  padding-inline: var(--space-sm);
  font: inherit;
  transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background var(--dur-quick) var(--ease-standard);
}
[data-rcl-input]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted); }
[data-rcl-input]:hover:not(:disabled) { border-color: var(--color-primary); }
[data-rcl-input]:focus-visible { border-color: var(--color-focus); outline: var(--border-strong) solid color-mix(in srgb, var(--color-focus) 30%, transparent); outline-offset: var(--space-3xs); }
[data-rcl-input]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }
@media (prefers-reduced-motion: reduce) { [data-rcl-input] { transition: none; } }

/* The "mobile-safe font sizing" this component has always advertised, finally
   implemented. iOS zooms the whole viewport when a text control smaller than
   16px receives focus, which on a fixed-position layout is a one-way trip.
   Deliberately at the component's own weight, so a consumer that states a size
   still wins: the promise is that the *default* is safe, not that the library
   overrules a product decision. '1em' resolves against the parent, so this is
   a floor and never an enlargement. */
@media (pointer: coarse) {
  [data-rcl-input] { font-size: max(16px, 1em); }
}
`;

/**
 * 1.2.0 moves style injection from a raw `<style>` element rendered beside the
 * control to the shared sheet registry. The rules are byte-identical; what
 * changes is that a page with N inputs now has one sheet in `<head>` instead
 * of N duplicated `<style>` nodes interleaved through the body, and the sheet
 * lands ahead of consumer styles the way every other library asset's does.
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, type, "data-testid": testID, ...props },
  ref,
) {
  return (
    <>
      <StyleSheet name="input-1-3-0" css={styleSheet} />
      <input
        ref={ref}
        type={type}
        data-testid={testID ?? "forms.input"}
        data-rcl-input="true"
        className={className}
        {...props}
      />
    </>
  );
});
