/** @vrooliComponentSource forms.validation-summary */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import {
  useEffect,
  useId,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { type FormStore } from "../../../../services/FormStore/versions/1.0.0/FormStore";

export interface ValidationSummaryProps<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  store?: FormStore<TValues>;
  errors?: Partial<Record<keyof TValues, string | undefined>>;
  fieldLabels?: Partial<Record<keyof TValues, ReactNode>>;
  title?: ReactNode;
  onFocusField?: (field: keyof TValues) => void;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-validation-summary] { display: grid; gap: var(--space-xs, .625rem); padding: var(--space-md, 1rem); border: 1px solid color-mix(in srgb, var(--color-danger, #dc2626) 45%, var(--color-border, #cbd5e1)); border-radius: var(--radius-control, .625rem); background: color-mix(in srgb, var(--color-danger, #dc2626) 7%, var(--color-surface, #fff)); color: var(--color-foreground, #0f172a); }
  [data-rcl-validation-summary-title] { display: flex; align-items: center; gap: var(--space-xs, .625rem); color: var(--color-danger, #dc2626); font: var(--text-label, 600 .8125rem/1.25rem system-ui, sans-serif); }
  [data-rcl-validation-summary-mark] { display: inline-grid; place-items: center; flex: 0 0 auto; inline-size: 1.25rem; block-size: 1.25rem; border: 1px solid currentColor; border-radius: 50%; font: 700 .75rem/1 system-ui, sans-serif; }
  [data-rcl-validation-summary-list] { display: grid; gap: var(--space-3xs, .25rem); margin: 0; padding-inline-start: calc(1.25rem + var(--space-xs, .625rem)); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-validation-summary-list] a { color: var(--color-danger, #dc2626); text-decoration-thickness: .1em; text-underline-offset: .15em; }
  [data-rcl-validation-summary-list] a:focus-visible { outline: 2px solid var(--color-focus, #2563eb); outline-offset: 3px; border-radius: .125rem; }
`;

function useFormSnapshot<TValues extends Record<string, unknown>>(
  store?: FormStore<TValues>,
) {
  const [, rerender] = useState(0);
  useEffect(() => {
    if (!store) return;
    return store.subscribe(() => rerender((count) => count + 1));
  }, [store]);
  return store?.get();
}

export function ValidationSummary<
  TValues extends Record<string, unknown> = Record<string, unknown>,
>({
  store,
  errors,
  fieldLabels,
  title = translate("forms.validation-summary.title.1", "Review these fields"),
  onFocusField,
  className,
  style,
}: ValidationSummaryProps<TValues>) {
  const state = useFormSnapshot(store);
  const headingId = useId().replace(/:/g, "");
  const stateErrors = state
    ? (Object.fromEntries(
        (
          Object.entries(state.fields) as Array<
            [keyof TValues, { error?: string }]
          >
        )
          .filter(([, field]) => field.error)
          .map(([field, value]) => [field, value.error]),
      ) as Partial<Record<keyof TValues, string | undefined>>)
    : {};
  const fieldErrors = errors ?? stateErrors;
  const entries = (
    Object.entries(fieldErrors) as Array<[keyof TValues, string | undefined]>
  ).filter(([, message]) => Boolean(message));
  if (entries.length === 0) return null;
  return (
    <>
      <style
        data-rcl-validation-summary-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <section
        className={className}
        style={style}
        data-rcl-validation-summary="true"
        role="alert"
        aria-labelledby={headingId}
      >
        <div id={headingId} data-rcl-validation-summary-title>
          <span data-rcl-validation-summary-mark aria-hidden="true">
            !
          </span>
          <span>{title}</span>
        </div>
        <ul data-rcl-validation-summary-list>
          {entries.map(([field, message]) => (
            <li key={String(field)}>
              <a data-testid="forms.validation-summary"
                href={`#${String(field)}`}
                onClick={() => onFocusField?.(field)}
              >
                {fieldLabels?.[field] ?? String(field)}
              </a>
              {": "} {message}
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
