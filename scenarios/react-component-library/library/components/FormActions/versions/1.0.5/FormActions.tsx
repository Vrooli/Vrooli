/**
 * @libraryId react-component-library:FormActions
 * @displayName FormActions
 * @description The coordinated action region for submit, cancel, reset, and delete, with save status, responsive placement, optional sticky behavior, and awareness of the form's async state.
 * @version 1.0.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:FormActions
 * @vrooliComponentSourceSlot forms.form-actions */
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { type FormPhase, type FormStore } from "@vrooli/react-component-library/FormStore/1";

export interface FormActionsProps<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  store?: FormStore<TValues>;
  submitLabel?: ReactNode;
  pendingLabel?: ReactNode;
  resetLabel?: ReactNode;
  cancelLabel?: ReactNode;
  onCancel?: () => void;
  children?: ReactNode;
  align?: "start" | "center" | "end" | "between";
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-form-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs); }
  [data-rcl-form-actions][data-align="start"] { justify-content: flex-start; }
  [data-rcl-form-actions][data-align="center"] { justify-content: center; }
  [data-rcl-form-actions][data-align="end"] { justify-content: flex-end; }
  [data-rcl-form-actions][data-align="between"] { justify-content: space-between; }
  [data-rcl-form-action] { min-block-size: 2.625rem; padding-inline: var(--space-md); border: 1px solid transparent; border-radius: var(--radius-control); font: var(--text-label); cursor: pointer; transition: background-color 160ms ease, border-color 160ms ease, box-shadow 160ms ease, opacity 160ms ease; }
  [data-rcl-form-action="submit"] { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-primary-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-form-action="submit"]:hover:not(:disabled) { filter: brightness(1.08); }
  [data-rcl-form-action="reset"], [data-rcl-form-action="cancel"] { border-color: var(--color-border-strong); background: var(--color-surface); color: var(--color-foreground); }
  [data-rcl-form-action="reset"]:hover:not(:disabled), [data-rcl-form-action="cancel"]:hover:not(:disabled) { background: var(--color-surface-raised); }
  [data-rcl-form-action]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }
  @media (max-width: 34rem) { [data-rcl-form-actions] { align-items: stretch; flex-direction: column-reverse; } [data-rcl-form-action] { inline-size: 100%; } }
`;

const phaseBusy = (phase: FormPhase | undefined) =>
  phase === "validating" || phase === "submitting" || phase === "saving";

function useFormPhase<TValues extends Record<string, unknown>>(store?: FormStore<TValues>) {
  const [, rerender] = useState(0);
  useEffect(() => {
    if (!store) return;
    return store.subscribe(() => rerender((count) => count + 1));
  }, [store]);
  return store?.get().phase;
}

export const FormActions = withClassName(function FormActions<
  TValues extends Record<string, unknown> = Record<string, unknown>,
>({
  store,
  submitLabel = "Save changes",
  pendingLabel = "Saving…",
  resetLabel,
  cancelLabel,
  onCancel,
  children,
  align = "end",
  disabled = false,
  className,
  style,
}: FormActionsProps<TValues>) {
  const phase = useFormPhase(store);
  const busy = phaseBusy(phase);
  const isDisabled = disabled || busy;
  return (
    <>
      <StyleSheet libraryId="react-component-library:FormActions" version="1.0.4" css={styles} />
      <div
        className={className}
        style={style}
        data-rcl-form-actions="true"
        data-align={align}
        data-phase={phase ?? "idle"}
      >
        {children ?? (
          <>
            {cancelLabel && (
              <button
                data-testid="forms.form-actions"
                type="button"
                data-rcl-form-action="cancel"
                onClick={onCancel}
                disabled={isDisabled}
              >
                {cancelLabel}
              </button>
            )}
            {resetLabel && (
              <button
                data-testid="forms.form-actions"
                type="reset"
                data-rcl-form-action="reset"
                disabled={isDisabled}
              >
                {resetLabel}
              </button>
            )}
            <button
              data-testid="forms.form-actions"
              type="submit"
              data-rcl-form-action="submit"
              disabled={isDisabled}
              aria-busy={busy || undefined}
            >
              {busy ? pendingLabel : submitLabel}
            </button>
          </>
        )}
      </div>
    </>
  );
});
