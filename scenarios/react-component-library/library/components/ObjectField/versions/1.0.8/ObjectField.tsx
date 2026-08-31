/**
 * @libraryId react-component-library:ObjectField
 * @displayName ObjectField
 * @description The nested-object field grouping child fields, collapsing sections, summarizing nested errors, supporting optional objects, and preserving per-field subscriptions.
 * @version 1.0.8
 * @tags ["form","object","nested","validation","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource forms.object-field */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { FormSection } from "@vrooli/react-component-library/FormSection/1";
import type { FormStore } from "@vrooli/react-component-library/FormStore/1";

export interface ObjectFieldContext<TObject extends Record<string, unknown>> {
  value: TObject;
  setValue: <K extends keyof TObject>(key: K, value: TObject[K]) => void;
  getError: (key: keyof TObject) => string | undefined;
}

export interface ObjectFieldProps<
  TValues extends Record<string, unknown>,
  K extends keyof TValues,
  TObject extends Record<string, unknown> = TValues[K] extends Record<string, unknown>
    ? TValues[K]
    : Record<string, unknown>,
> {
  store: FormStore<TValues>;
  field: K;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode | ((context: ObjectFieldContext<TObject>) => ReactNode);
  errors?: Partial<Record<keyof TObject, string>>;
  defaultValue?: TObject;
  collapsible?: boolean;
  defaultOpen?: boolean;
  optional?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-object-field] { min-inline-size: 0; }
  [data-rcl-object-content] { display: grid; gap: var(--space-md, 24px); min-inline-size: 0; }
  [data-rcl-object-meta] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs, 8px); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-object-badge] { display: inline-flex; align-items: center; min-block-size: 1.5rem; padding-inline: var(--space-2xs, 8px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-pill, 9999px); background: var(--color-surface, #ffffff); }
  @media (max-width: 34rem) { [data-rcl-object-meta] { align-items: flex-start; flex-direction: column; } }
`;

export const ObjectField = withClassName(function ObjectField<
  TValues extends Record<string, unknown>,
  K extends keyof TValues,
  TObject extends Record<string, unknown> = TValues[K] extends Record<string, unknown>
    ? TValues[K]
    : Record<string, unknown>,
>({
  store,
  field,
  title,
  description,
  children,
  errors = {},
  defaultValue = {} as TObject,
  collapsible = true,
  defaultOpen = true,
  optional = false,
  className,
  style,
}: ObjectFieldProps<TValues, K, TObject>) {
  const libraryStrings = useStrings();
  const [, rerender] = useState(0);
  useEffect(() => store.subscribe(() => rerender((count) => count + 1)), [store]);
  const fieldState = store.getField(field);
  const value = (
    fieldState.value && typeof fieldState.value === "object" && !Array.isArray(fieldState.value)
      ? fieldState.value
      : defaultValue
  ) as TObject;
  const nestedErrorCount = Object.values(errors).filter(Boolean).length;
  const rootError = fieldState.error;
  const setValue = <TKey extends keyof TObject>(key: TKey, next: TObject[TKey]) => {
    store.setValue(field, { ...value, [key]: next } as TValues[K]);
  };
  const context: ObjectFieldContext<TObject> = {
    value,
    setValue,
    getError: (key) => errors[key],
  };
  const content = typeof children === "function" ? children(context) : children;

  return (
    <div
      data-testid="forms.object-field"
      className={className}
      style={style}
      data-rcl-object-field
      data-field={String(field)}
    >
      <StyleSheet name="object-field-1-0-4" css={styles} />
      <FormSection
        title={title}
        description={description}
        errorCount={nestedErrorCount + (rootError ? 1 : 0)}
        collapsible={collapsible}
        defaultOpen={defaultOpen}
        summary={
          <span data-rcl-object-meta>
            <span data-rcl-object-badge>{Object.keys(value).length} values</span>
            {optional && (
              <span>{libraryStrings("forms.object-field.optional-group", "Optional group")}</span>
            )}
          </span>
        }
      >
        <div data-rcl-object-content>
          {rootError && (
            <div role="alert" data-rcl-object-meta>
              {rootError}
            </div>
          )}
          {content}
        </div>
      </FormSection>
    </div>
  );
});
