/**
 * @libraryId react-component-library:Select
 * @displayName Select
 * @description Native select wrapper styled with Vrooli tokens and mobile-safe control sizing.
 * @version 1.2.0
 * @tags ["form","interactive"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
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
[data-rcl-select] { box-sizing: border-box; inline-size: 100%; min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-field, var(--color-surface)); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-select]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-select][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-select]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }


/* The "mobile-safe font sizing" this component has always advertised, finally
   implemented. iOS zooms the whole viewport when a text control smaller than
   16px receives focus, which on a fixed-position layout is a one-way trip.
   Deliberately at the component's own weight, so a consumer that states a size
   still wins: the promise is that the *default* is safe, not that the library
   overrules a product decision. '1em' resolves against the parent, so this is
   a floor and never an enlargement. */
@media (pointer: coarse) {
  [data-rcl-select] { font-size: max(16px, 1em); }
}
`;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className, options, placeholder, ...props },
  ref,
) {
  return (
    <>
      <StyleSheet name="select-1-2-0-1" css={styleSheet} />
      <select
        data-testid="forms.select"
        ref={ref}
        data-rcl-select="true"
        className={className}
        {...props}
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
    </>
  );
});
