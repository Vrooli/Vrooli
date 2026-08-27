/**
 * @libraryId react-component-library:Input
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { forwardRef, type InputHTMLAttributes } from "react";
export const INPUT_MODES = ["controlled", "uncontrolled"] as const;
export const INPUT_SIZES = ["sm", "md", "lg"] as const;
export const INPUT_TONES = ["default", "invalid"] as const;
export const INPUT_PARTS = ["prefix", "control", "suffix"] as const;

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  "data-testid"?: string;
};

const joinClasses = (...inputs: Array<string | undefined>) =>
  inputs.filter(Boolean).join(" ");

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
[data-rcl-input]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }

`;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, type, "data-testid": testID, ...props },
  ref,
) {
  return (
    <>
      <StyleSheet name="input-1-1-0-1" css={styleSheet} />
      <input
        ref={ref}
        type={type}
        data-testid={testID ?? "rcl-input"}
        data-rcl-input="true"
        className={joinClasses(
          "rounded-control border border-app-border",
          className,
        )}
        {...props}
      />
    </>
  );
});
