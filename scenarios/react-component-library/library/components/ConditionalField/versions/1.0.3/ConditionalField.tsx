/**
 * @libraryId react-component-library:ConditionalField
 * @displayName ConditionalField
 * @version 1.0.3
 * @tags ["form","conditional","state","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource forms.conditional-field */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { type FormStore } from "@vrooli/react-component-library/FormStore/1";

export type ConditionalFieldMode = "hide" | "disable" | "reset";

export interface ConditionalFieldProps<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  store: FormStore<TValues>;
  field: keyof TValues;
  when: (values: TValues) => boolean;
  children: ReactNode;
  fallback?: ReactNode;
  mode?: ConditionalFieldMode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-conditional-field] { min-inline-size: 0; transition: opacity var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
  [data-rcl-conditional-field][data-disabled="true"] { opacity: .56; }
  [data-rcl-conditional-field][data-disabled="true"] > * { pointer-events: none; }

`;

function useStoreValues<TValues extends Record<string, unknown>>(store: FormStore<TValues>) {
  const [, rerender] = useState(0);
  useEffect(() => store.subscribe(() => rerender((count) => count + 1)), [store]);
  return store.getValues();
}

export const ConditionalField = withClassName(function ConditionalField<
  TValues extends Record<string, unknown>,
>({
  store,
  field,
  when,
  children,
  fallback = null,
  mode = "hide",
  className,
  style,
}: ConditionalFieldProps<TValues>) {
  const values = useStoreValues(store);
  const visible = when(values);

  useEffect(() => {
    if (!visible && mode === "reset") {
      const current = store.getField(field);
      if (!Object.is(current.value, current.defaultValue))
        store.setValue(field, current.defaultValue);
    }
  }, [field, mode, store, visible]);

  if (!visible && (mode === "hide" || mode === "reset")) return <>{fallback}</>;
  return (
    <div
      data-testid="forms.conditional-field"
      className={className}
      style={style}
      data-rcl-conditional-field
      data-visible={visible}
      data-disabled={!visible && mode === "disable"}
      aria-hidden={(!visible && mode === "hide") || undefined}
    >
      <StyleSheet name="conditionalfield-1-0-1-1" css={styles} />
      {children}
    </div>
  );
});
