/** @vrooliComponentSource forms.conditional-field */
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { type FormStore } from "../../../../services/FormStore/versions/1.0.0/FormStore";

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
  [data-rcl-conditional-field] { min-inline-size: 0; transition: opacity var(--dur-quick, 160ms) var(--ease-standard, ease); }
  [data-rcl-conditional-field][data-disabled="true"] { opacity: .56; }
  [data-rcl-conditional-field][data-disabled="true"] > * { pointer-events: none; }
  @media (prefers-reduced-motion: reduce) { [data-rcl-conditional-field] { transition: none; } }
`;

function useStoreValues<TValues extends Record<string, unknown>>(
  store: FormStore<TValues>,
) {
  const [, rerender] = useState(0);
  useEffect(
    () => store.subscribe(() => rerender((count) => count + 1)),
    [store],
  );
  return store.getValues();
}

export function ConditionalField<TValues extends Record<string, unknown>>({
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
      className={className}
      style={style}
      data-rcl-conditional-field
      data-visible={visible}
      data-disabled={!visible && mode === "disable"}
      aria-hidden={(!visible && mode === "hide") || undefined}
    >
      <style
        data-rcl-conditional-field-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      {children}
    </div>
  );
}
