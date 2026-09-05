/**
 * @libraryId react-component-library:Form
 * @displayName Form
 * @version 1.0.3
 * @tags ["form","state","accessibility","compound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource forms.form */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  useEffect,
  useId,
  useState,
  type CSSProperties,
  type FormEvent,
  type FormHTMLAttributes,
  type ReactNode,
} from "react";
import { type FormPhase, type FormStore } from "@vrooli/react-component-library/FormStore/1";

export type FormValues = Record<string, unknown>;

export type FormMode = "controlled" | "uncontrolled";

export type FormSubmitHandler<TValues extends FormValues> = (
  values: TValues,
) => void | Promise<void>;

export interface FormProps<TValues extends FormValues = FormValues>
  extends Omit<FormHTMLAttributes<HTMLFormElement>, "onSubmit" | "onReset" | "title"> {
  mode?: FormMode;
  store?: FormStore<TValues>;
  onSubmit?: FormSubmitHandler<TValues>;
  onReset?: () => void;
  children: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  footer?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-form] { display: grid; gap: var(--space-lg, 32px); color: var(--color-foreground, #0f172a); }
  [data-rcl-form-header] { display: grid; gap: var(--space-2xs, 8px); }
  [data-rcl-form-title] { color: var(--color-foreground, #0f172a); font: var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans)); letter-spacing: var(--text-title-tracking, -.01em); }
  [data-rcl-form-description] { max-inline-size: 60ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  [data-rcl-form-body] { display: grid; gap: var(--space-md, 24px); min-inline-size: 0; }
  [data-rcl-form-footer] { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-sm, 16px); padding-block-start: var(--space-sm, 16px); border-block-start: 1px solid var(--color-border, #cbd5e1); }
  [data-rcl-form-status] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-form-status][data-phase="error"] { color: var(--color-danger, #dc2626); }
  [data-rcl-form-status][data-phase="success"] { color: var(--color-success, #16a34a); }
  [data-rcl-form-conflict] { display: flex; align-items: flex-start; gap: var(--space-2xs, 8px); padding: var(--space-sm, 16px); border: 1px solid color-mix(in srgb, var(--color-warning, #d97706) 42%, var(--color-border, #cbd5e1)); border-radius: var(--radius-control, 0.375rem); background: color-mix(in srgb, var(--color-warning, #d97706) 9%, var(--color-surface, #ffffff)); color: var(--color-foreground, #0f172a); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  [data-rcl-form-conflict-mark] { display: inline-grid; place-items: center; flex: 0 0 auto; inline-size: 1.125rem; block-size: 1.125rem; margin-block-start: .125rem; border: 1px solid currentColor; border-radius: 50%; color: var(--color-warning, #d97706); font: 700 .6875rem/1 system-ui, sans-serif; }
`;

const phaseMessage: Record<FormPhase, string> = {
  idle: "Ready to save.",
  validating: "Checking your changes…",
  submitting: "Saving your changes…",
  saving: "Saving a draft…",
  success: "Changes saved.",
  error: "Review the highlighted fields and try again.",
  offline: "Changes are queued until you reconnect.",
};

function useFormState<TValues extends FormValues>(store?: FormStore<TValues>) {
  const [, rerender] = useState(0);
  useEffect(() => {
    if (!store) return;
    return store.subscribe(() => rerender((count) => count + 1));
  }, [store]);
  return store?.get();
}

export const Form = withClassName(function Form<TValues extends FormValues = FormValues>({
  mode,
  store,
  onSubmit,
  onReset,
  children,
  title,
  description,
  footer,
  className,
  style,
  id,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: FormProps<TValues>) {
  const resolvedMode: FormMode = mode ?? (store ? "controlled" : "uncontrolled");
  const state = useFormState(store);
  const generatedStatusId = useId().replace(/:/g, "");
  const statusId = `${id ?? `rcl-form-${generatedStatusId}`}-status`;
  const phase = state?.phase ?? "idle";
  const busy = phase === "validating" || phase === "submitting" || phase === "saving";
  const status = state?.error ?? phaseMessage[phase];
  const hasConflict = Boolean(state?.conflict);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (store && onSubmit) {
      await store.submit(async (values) => {
        await onSubmit(values);
        return undefined;
      });
      return;
    }
    if (store) await store.validate();
    else if (onSubmit) void onSubmit({} as TValues);
  };

  const reset = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    store?.reset();
    onReset?.();
  };

  return (
    <>
      <StyleSheet name="form-1-0-1" css={styles} />
      <form
        data-testid="forms.form"
        {...props}
        id={id}
        className={className}
        style={style}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        aria-describedby={statusId}
        aria-busy={busy || undefined}
        data-rcl-form="true"
        data-form-mode={resolvedMode}
        data-phase={phase}
        onSubmit={(event) => void submit(event)}
        onReset={reset}
      >
        {(title || description) && (
          <header data-rcl-form-header>
            {title && <div data-rcl-form-title>{title}</div>}
            {description && <div data-rcl-form-description>{description}</div>}
          </header>
        )}
        {hasConflict && (
          <div data-rcl-form-conflict role="alert">
            <span data-rcl-form-conflict-mark aria-hidden="true">
              !
            </span>
            <span>{state?.conflict?.message}</span>
          </div>
        )}
        <div data-rcl-form-body>{children}</div>
        <div data-rcl-form-footer>
          <div
            id={statusId}
            data-rcl-form-status
            data-phase={phase}
            role="status"
            aria-live="polite"
          >
            {status}
          </div>
          {footer}
        </div>
      </form>
    </>
  );
});
