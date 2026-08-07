/** @vrooliComponentSource forms.form-actions */
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import {
  type FormPhase,
  type FormStore,
} from "../../../../services/FormStore/versions/1.0.0/FormStore";

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
  [data-rcl-form-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs, .625rem); }
  [data-rcl-form-actions][data-align="start"] { justify-content: flex-start; }
  [data-rcl-form-actions][data-align="center"] { justify-content: center; }
  [data-rcl-form-actions][data-align="end"] { justify-content: flex-end; }
  [data-rcl-form-actions][data-align="between"] { justify-content: space-between; }
  [data-rcl-form-action] { min-block-size: 2.625rem; padding-inline: var(--space-md, 1rem); border: 1px solid transparent; border-radius: var(--radius-control, .625rem); font: var(--text-label, 600 .8125rem/1.25rem system-ui, sans-serif); cursor: pointer; transition: background-color 160ms ease, border-color 160ms ease, box-shadow 160ms ease, opacity 160ms ease; }
  [data-rcl-form-action]:focus-visible { outline: 2px solid var(--color-focus, #2563eb); outline-offset: 3px; }
  [data-rcl-form-action="submit"] { border-color: var(--color-primary, #2563eb); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #fff); box-shadow: var(--elev-raised, 0 4px 12px rgb(15 23 42 / .12)); }
  [data-rcl-form-action="submit"]:hover:not(:disabled) { filter: brightness(1.08); }
  [data-rcl-form-action="reset"], [data-rcl-form-action="cancel"] { border-color: var(--color-border-strong, #94a3b8); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); }
  [data-rcl-form-action="reset"]:hover:not(:disabled), [data-rcl-form-action="cancel"]:hover:not(:disabled) { background: var(--color-surface-raised, #f8fafc); }
  [data-rcl-form-action]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled, .58); }
  @media (max-width: 34rem) { [data-rcl-form-actions] { align-items: stretch; flex-direction: column-reverse; } [data-rcl-form-action] { inline-size: 100%; } }
`;

const phaseBusy = (phase: FormPhase | undefined) =>
  phase === "validating" || phase === "submitting" || phase === "saving";

function useFormPhase<TValues extends Record<string, unknown>>(
  store?: FormStore<TValues>,
) {
  const [, rerender] = useState(0);
  useEffect(() => {
    if (!store) return;
    return store.subscribe(() => rerender((count) => count + 1));
  }, [store]);
  return store?.get().phase;
}

export function FormActions<
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
      <style
        data-rcl-form-actions-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
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
                type="reset"
                data-rcl-form-action="reset"
                disabled={isDisabled}
              >
                {resetLabel}
              </button>
            )}
            <button
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
}
