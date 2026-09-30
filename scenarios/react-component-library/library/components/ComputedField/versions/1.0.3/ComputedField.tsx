/**
 * @libraryId react-component-library:ComputedField
 * @displayName ComputedField
 * @version 1.0.3
 * @tags ["form","computed","derived","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource forms.computed-field */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { type FormStore } from "@vrooli/react-component-library/FormStore/1";

export interface ComputedFieldProps<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  store: FormStore<TValues>;
  field: keyof TValues;
  compute: (values: TValues) => TValues[keyof TValues];
  label: ReactNode;
  description?: ReactNode;
  format?: (value: TValues[keyof TValues]) => ReactNode;
  override?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-computed-field] { display: grid; gap: var(--space-2xs); min-inline-size: 0; color: var(--color-foreground); }
  [data-rcl-computed-label] { color: var(--color-foreground); font: var(--text-label); }
  [data-rcl-computed-output] { display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm); min-block-size: 2.75rem; box-sizing: border-box; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-muted)); padding: .625rem .875rem; font: var(--text-body); }
  [data-rcl-computed-value] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 700; }
  [data-rcl-computed-badge] { flex: 0 0 auto; color: var(--color-primary); font: var(--text-caption); }
  [data-rcl-computed-description] { color: var(--color-muted-foreground); font: var(--text-body); }
`;

function useStoreValues<TValues extends Record<string, unknown>>(store: FormStore<TValues>) {
  const [, rerender] = useState(0);
  useEffect(() => store.subscribe(() => rerender((count) => count + 1)), [store]);
  return store.getValues();
}

export const ComputedField = withClassName(function ComputedField<
  TValues extends Record<string, unknown>,
>({
  store,
  field,
  compute,
  label,
  description,
  format = (value) => String(value),
  override = false,
  className,
  style,
}: ComputedFieldProps<TValues>) {
  const values = useStoreValues(store);
  const calculated = compute(values);
  const fieldState = store.getField(field);
  const labelText = typeof label === "string" ? label : "Calculated value";
  const value = override && fieldState.dirty ? fieldState.value : calculated;
  return (
    <div
      data-testid="forms.computed-field"
      className={className}
      style={style}
      data-rcl-computed-field
    >
      <StyleSheet libraryId="react-component-library:ComputedField" version="1.0.3" css={styles} />
      <span data-rcl-computed-label>{label}</span>
      <output
        data-rcl-computed-output
        data-rcl-computed-value
        aria-live="polite"
        aria-label={`${labelText} calculated value`}
      >
        <span data-rcl-computed-value>{format(value)}</span>
        <span data-rcl-computed-badge>
          {override && fieldState.dirty ? "Edited" : "Calculated"}
        </span>
      </output>
      {description && <span data-rcl-computed-description>{description}</span>}
    </div>
  );
});
