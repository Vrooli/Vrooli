/**
 * @libraryId react-component-library:Select
 * @displayName Select
 * @description The single-selection field with native-like semantics, keyboard navigation, option groups, descriptions, async option loading, and responsive popover or sheet presentation.
 * @version 1.3.1
 * @tags []
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
  /** Value-only change callback for composed controls and typed adapters. */
  onValueChange?: (value: string) => void;
}

const styleSheet = `
[data-rcl-select-field] { position: relative; display: block; min-inline-size: 0; }
[data-rcl-select] { box-sizing: border-box; inline-size: 100%; min-block-size: var(--tap-target-min); appearance: none; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-field); color: var(--color-foreground); padding: var(--space-2xs) var(--space-lg) var(--space-2xs) var(--space-sm); font: inherit; cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-select-field]::after { position: absolute; inset-inline-end: var(--space-sm); inset-block-start: 50%; inline-size: .45rem; block-size: .45rem; border-inline-end: var(--border-strong) solid currentColor; border-block-end: var(--border-strong) solid currentColor; color: var(--color-muted-foreground); content: ""; pointer-events: none; transform: translateY(-65%) rotate(45deg); }
[data-rcl-select]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-select]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
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
  { className, options, placeholder, onChange, onValueChange, ...props },
  ref,
) {
  return (
    <>
      <StyleSheet
        libraryId="react-component-library:Select"
        version="1.3.1-draft.1"
        css={styleSheet}
      />
      <span data-rcl-select-field>
        <select
          data-testid="forms.select"
          ref={ref}
          data-rcl-select="true"
          className={className}
          {...props}
          onChange={(event) => {
            onChange?.(event);
            onValueChange?.(event.currentTarget.value);
          }}
        >
          {placeholder && <option value="">{placeholder}</option>}
          {options.map((option) => (
            <option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </option>
          ))}
        </select>
      </span>
    </>
  );
});
