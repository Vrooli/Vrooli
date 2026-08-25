/**
 * @libraryId react-component-library:Select
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import { forwardRef, type SelectHTMLAttributes } from "react";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[];
  placeholder?: string;
}

const styleSheet = `
[data-rcl-select] { box-sizing: border-box; inline-size: 100%; min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-select]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-select]:focus-visible { border-color: var(--color-focus); outline: var(--border-strong) solid color-mix(in srgb, var(--color-focus) 30%, transparent); outline-offset: var(--space-3xs); }
[data-rcl-select][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-select]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }
@media (prefers-reduced-motion: reduce) { [data-rcl-select] { transition: none; } }
`;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select({ className, options, placeholder, ...props }, ref) {
    return (
      <>
        <style
          data-rcl-select-styles
          dangerouslySetInnerHTML={{ __html: styleSheet }}
        />
        <select data-testid="forms.select"
          ref={ref}
          data-rcl-select="true"
          className={className}
          {...props}
        >
          {placeholder && <option value="">{placeholder}</option>}
          {options.map((option) => (
            <option
              key={option.value}
              value={option.value}
              disabled={option.disabled}
            >
              {option.label}
            </option>
          ))}
        </select>
      </>
    );
  },
);
