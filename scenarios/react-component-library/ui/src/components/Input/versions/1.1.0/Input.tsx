/**
 * @vrooliComponentSource react-component-library:Input
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption cbda07b5-f3f4-48cb-b391-b70d59a6df1d
 * @vrooliComponentAppliedAt 2026-08-17T23:22:44Z
 * @vrooliComponentSourceSha256 9f252ad6fdee171d8c67d6b5efb9461b2fa4ad3393e6483ea95a7b3de92832b1
 * @vrooliComponentDriftHash 031b0ea19609b00d266da0d78be9f6e9337b6bea7da30d63828b406c5f83390a
 * @vrooliComponentTokenTranslation border-app-border->border-app-border
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef, type InputHTMLAttributes } from "react";
export const INPUT_MODES = ["controlled", "uncontrolled"] as const;
export const INPUT_SIZES = ["sm", "md", "lg"] as const;
export const INPUT_TONES = ["default", "invalid"] as const;
export const INPUT_PARTS = ["prefix", "control", "suffix"] as const;

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  "data-testid"?: string;
};

const joinClasses = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

const styleSheet = `
[data-rcl-input] {
  box-sizing: border-box;
  width: 100%;
  min-height: var(--tap-target-min);
  border: var(--border-hairline) solid var(--color-border);
  border-radius: var(--radius-control);
  background: var(--color-surface);
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
`;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, type, "data-testid": testID, ...props },
  ref,
) {
  return (
    <>
      <style data-rcl-input-styles dangerouslySetInnerHTML={{ __html: styleSheet }} />
      <input
        ref={ref}
        type={type}
        data-testid={testID ?? "rcl-input"}
        data-rcl-input="true"
        className={joinClasses("rounded-control border border-app-border", className)}
        {...props}
      />
    </>
  );
});
