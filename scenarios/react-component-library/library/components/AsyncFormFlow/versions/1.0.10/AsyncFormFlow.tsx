/**
 * @libraryId react-component-library:AsyncFormFlow
 * @displayName AsyncFormFlow
 * @description The create-or-edit workflow covering initial load, validation, submission, progress, server-error mapping, success, retry, cancellation, and the navigation that follows.
 * @version 1.0.10
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:AsyncFormFlow
 * @vrooliComponentSourceSlot patterns.async-form-flow */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import {
  AsyncBoundary,
  type AsyncBoundaryStatus,
} from "@vrooli/react-component-library/AsyncBoundary/1";
import { Form, type FormValues } from "@vrooli/react-component-library/Form/1";
import { FormActions } from "@vrooli/react-component-library/FormActions/1";
import { ValidationSummary } from "@vrooli/react-component-library/ValidationSummary/1";
import {
  createFormStore,
  type FormStore,
  type FormValidationResult,
} from "@vrooli/react-component-library/FormStore/1";

export type AsyncFormLoadState =
  | "idle"
  | "loading"
  | "refreshing"
  | "empty"
  | "partial"
  | "stale"
  | "success"
  | "request-error"
  | "offline";

export interface AsyncFormLoadResult<TValues extends FormValues> {
  values?: Partial<TValues>;
  state?: "empty" | "partial" | "success";
}

export interface AsyncFormSubmitError<TValues extends FormValues> extends Error {
  fieldErrors?: Partial<Record<keyof TValues, string>>;
}

export interface AsyncFormFlowContext<TValues extends FormValues> {
  store: FormStore<TValues>;
  loadState: AsyncFormLoadState;
  submitting: boolean;
  cancelSubmit: () => void;
  refresh: () => void;
}

export interface AsyncFormFlowProps<TValues extends FormValues = FormValues> {
  initialValues: TValues;
  children: ReactNode | ((context: AsyncFormFlowContext<TValues>) => ReactNode);
  load?: (
    signal: AbortSignal,
  ) => AsyncFormLoadResult<TValues> | Promise<AsyncFormLoadResult<TValues>>;
  validate?: (
    values: TValues,
  ) => FormValidationResult<TValues> | Promise<FormValidationResult<TValues>>;
  onSubmit: (values: TValues, signal: AbortSignal) => void | Promise<void>;
  onNavigate?: (destination?: string) => void;
  destination?: string;
  offline?: boolean;
  title?: ReactNode;
  description?: ReactNode;
  successMessage?: ReactNode;
  errorMessage?: ReactNode;
  submitLabel?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-async-form-flow] { min-inline-size: 0; overflow: clip; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-async-form-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-md); padding: var(--space-lg) var(--space-lg) 0; }
  [data-rcl-async-form-kicker] { color: var(--color-primary); font: 800 .6875rem/1.2 system-ui, sans-serif; letter-spacing: .13em; text-transform: uppercase; }
  [data-rcl-async-form-title] { margin-block-start: var(--space-2xs); font: var(--text-title); letter-spacing: -.02em; }
  [data-rcl-async-form-description] { max-inline-size: 52ch; margin-block-start: var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-async-form-refresh] { flex: 0 0 auto; min-block-size: var(--tap-target-min); border: 1px solid var(--color-border-strong); border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-label); cursor: pointer; }
  [data-rcl-async-form-refresh]:hover { background: var(--color-surface-raised); }
  [data-rcl-async-form-content] { padding: var(--space-lg); }
  [data-rcl-async-form-empty] { display: grid; gap: var(--space-xs); place-items: start; padding: var(--space-lg); border: 1px dashed var(--color-border-strong); border-radius: var(--radius-control); background: var(--color-surface-muted); }
  [data-rcl-async-form-empty-title] { font: var(--text-subtitle); }
  [data-rcl-async-form-empty-copy] { color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-async-form-empty-action], [data-rcl-async-form-next] { min-block-size: var(--tap-target-min); border: 1px solid var(--color-primary); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); padding-inline: var(--space-md); font: var(--text-label); cursor: pointer; }
  [data-rcl-async-form-submit-status] { display: flex; align-items: flex-start; gap: var(--space-xs); margin-block-end: var(--space-md); padding: var(--space-sm) var(--space-md); border: 1px solid color-mix(in srgb, var(--color-success) 36%, var(--color-border)); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-success) 8%, var(--color-surface)); color: var(--color-success); font: var(--text-body); }
  [data-rcl-async-form-submit-status][data-phase="error"] { border-color: color-mix(in srgb, var(--color-danger) 38%, var(--color-border)); background: color-mix(in srgb, var(--color-danger) 7%, var(--color-surface)); color: var(--color-danger); }
  [data-rcl-async-form-submit-mark] { display: grid; flex: 0 0 auto; place-items: center; inline-size: 1.25rem; block-size: 1.25rem; border: 1px solid currentColor; border-radius: 50%; font: 800 .75rem/1 system-ui, sans-serif; }
  [data-rcl-async-form-submit-copy] { display: grid; gap: .125rem; min-inline-size: 0; }
  [data-rcl-async-form-submit-title] { color: var(--color-foreground); font-weight: 750; }
  [data-rcl-async-form-next] { margin-block-start: var(--space-sm); background: transparent; color: var(--color-primary); }
  [data-rcl-async-form-next]:hover { background: var(--color-surface-raised); }
  @media (max-width: 36rem) { [data-rcl-async-form-header] { display: grid; padding-inline: var(--space-md); } [data-rcl-async-form-refresh] { justify-self: start; } [data-rcl-async-form-content] { padding: var(--space-md); } }
`;

function boundaryStatus(state: AsyncFormLoadState): AsyncBoundaryStatus {
  switch (state) {
    case "loading":
      return "pending";
    case "refreshing":
      return "refreshing";
    case "partial":
      return "partial-error";
    case "stale":
      return "stale";
    case "offline":
      return "offline";
    case "request-error":
      return "error";
    default:
      return "success";
  }
}

function useFormSnapshot<TValues extends FormValues>(store: FormStore<TValues>) {
  const [, rerender] = useState(0);
  useEffect(() => store.subscribe(() => rerender((count) => count + 1)), [store]);
  return store.get();
}

export const AsyncFormFlow = withClassName(function AsyncFormFlow<
  TValues extends FormValues = FormValues,
>({
  initialValues,
  children,
  load,
  validate,
  onSubmit,
  onNavigate,
  destination,
  offline = false,
  title,
  description,
  successMessage = "Saved successfully. Your changes are safe.",
  errorMessage = "We could not save this change. Your input is still here; review the details and retry.",
  submitLabel = "Save changes",
  className,
  style,
}: AsyncFormFlowProps<TValues>) {
  const libraryStrings = useStrings();
  description =
    description ??
    libraryStrings(
      "patterns.async-form-flow.your-work-stays-in-place-while-we-validate-save-",
      "Your work stays in place while we validate, save, and recover from interruptions.",
    );
  title = title ?? libraryStrings("patterns.async-form-flow.create-or-edit", "Create or edit");
  const store = useMemo(
    () => createFormStore<TValues>({ initialValues, validate }),
    [initialValues, validate],
  );
  const [loadState, setLoadState] = useState<AsyncFormLoadState>(load ? "loading" : "success");
  const [submitCancelled, setSubmitCancelled] = useState(false);
  const activeLoad = useRef<AbortController>();
  const activeSubmit = useRef<AbortController>();
  const loaded = useRef(!load);

  const runLoad = () => {
    if (!load) return;
    activeLoad.current?.abort();
    const controller = new AbortController();
    activeLoad.current = controller;
    setLoadState(loaded.current ? "refreshing" : "loading");
    void Promise.resolve(load(controller.signal))
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.values) store.setValues(result.values);
        loaded.current = true;
        setLoadState(result.state ?? "success");
      })
      .catch(() => {
        if (!controller.signal.aborted) setLoadState("request-error");
      });
  };

  useEffect(() => {
    runLoad();
    return () => {
      activeLoad.current?.abort();
      activeSubmit.current?.abort();
    };
    // The store and load function are intentionally captured for one workflow instance.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const cancelSubmit = () => {
    setSubmitCancelled(true);
    activeSubmit.current?.abort();
    activeSubmit.current = undefined;
    store.setPhase("idle");
  };

  const submit = async (values: TValues) => {
    const controller = new AbortController();
    activeSubmit.current = controller;
    setSubmitCancelled(false);
    try {
      await onSubmit(values, controller.signal);
      if (!controller.signal.aborted && !submitCancelled) {
        onNavigate?.(destination);
      }
    } catch (error) {
      const typedError = error as AsyncFormSubmitError<TValues>;
      if (typedError.fieldErrors) {
        Object.entries(typedError.fieldErrors).forEach(([field, message]) => {
          if (message) store.setError(field, message);
        });
      }
      throw error;
    } finally {
      if (activeSubmit.current === controller) activeSubmit.current = undefined;
    }
    if (controller.signal.aborted || submitCancelled) store.setPhase("idle");
  };

  const phase = useFormSnapshot(store).phase;
  const submitting = phase === "validating" || phase === "submitting";
  const context: AsyncFormFlowContext<TValues> = {
    store,
    loadState,
    submitting,
    cancelSubmit,
    refresh: runLoad,
  };
  const body = typeof children === "function" ? children(context) : children;
  const showEmpty = loadState === "empty";
  const formError = store.get().error;
  const showSubmitError = phase === "error" && (Boolean(formError) || submitCancelled);

  return (
    <>
      <StyleSheet libraryId="react-component-library:AsyncFormFlow" version="1.0.9" css={styles} />
      <section className={className} style={style} data-rcl-async-form-flow>
        <header data-rcl-async-form-header>
          <div>
            <div data-rcl-async-form-kicker>
              {libraryStrings("patterns.async-form-flow.async-workflow", "ASYNC WORKFLOW")}
            </div>
            <div data-rcl-async-form-title>{title}</div>
            <div data-rcl-async-form-description>{description}</div>
          </div>
          {load && (
            <button
              data-testid="patterns.async-form-flow"
              type="button"
              data-rcl-async-form-refresh
              onClick={runLoad}
              disabled={loadState === "loading" || loadState === "refreshing"}
            >
              {loadState === "refreshing" ? "Refreshing…" : "Refresh data"}
            </button>
          )}
        </header>
        <AsyncBoundary
          status={boundaryStatus(loadState)}
          offline={offline}
          retry={runLoad}
          preserveContent={loaded.current && loadState !== "loading"}
          errorTitle="We could not load this form"
          error="The saved version is unavailable right now. Retry when the connection is stable."
        >
          <div data-rcl-async-form-content>
            {showEmpty ? (
              <div data-rcl-async-form-empty>
                <strong data-rcl-async-form-empty-title>
                  {libraryStrings(
                    "patterns.async-form-flow.no-saved-draft-yet",
                    "No saved draft yet",
                  )}
                </strong>
                <span data-rcl-async-form-empty-copy>
                  {libraryStrings(
                    "patterns.async-form-flow.start-a-fresh-version-or-retry-if-you-expected-a",
                    "Start a fresh version, or retry if you expected an existing draft.",
                  )}
                </span>
                <button
                  data-testid="patterns.async-form-flow"
                  type="button"
                  data-rcl-async-form-empty-action
                  onClick={() => setLoadState("success")}
                >
                  {libraryStrings("patterns.async-form-flow.start-fresh", "Start fresh")}
                </button>
              </div>
            ) : (
              <Form store={store} title={undefined} description={undefined} onSubmit={submit}>
                {showSubmitError && (
                  <div data-rcl-async-form-submit-status data-phase="error" role="alert">
                    <span data-rcl-async-form-submit-mark aria-hidden="true">
                      !
                    </span>
                    <span data-rcl-async-form-submit-copy>
                      <strong data-rcl-async-form-submit-title>
                        {libraryStrings(
                          "patterns.async-form-flow.save-needs-attention",
                          "Save needs attention",
                        )}
                      </strong>
                      <span>{formError ?? errorMessage}</span>
                    </span>
                  </div>
                )}
                {phase === "success" && (
                  <div data-rcl-async-form-submit-status data-phase="success" role="status">
                    <span data-rcl-async-form-submit-mark aria-hidden="true">
                      ✓
                    </span>
                    <span data-rcl-async-form-submit-copy>
                      <strong data-rcl-async-form-submit-title>
                        {libraryStrings("patterns.async-form-flow.all-set", "All set")}
                      </strong>
                      <span>{successMessage}</span>
                      {onNavigate && (
                        <button
                          data-testid="patterns.async-form-flow"
                          type="button"
                          data-rcl-async-form-next
                          onClick={() => onNavigate(destination)}
                        >
                          {libraryStrings("patterns.async-form-flow.continue", "Continue")}
                        </button>
                      )}
                    </span>
                  </div>
                )}
                <ValidationSummary store={store} />
                {body}
                <FormActions store={store} submitLabel={submitLabel} pendingLabel="Saving…">
                  <button
                    data-testid="patterns.async-form-flow"
                    type="button"
                    data-rcl-form-action="cancel"
                    onClick={cancelSubmit}
                    disabled={!submitting}
                  >
                    {libraryStrings("patterns.async-form-flow.cancel-request", "Cancel request")}
                  </button>
                  <button
                    data-testid="patterns.async-form-flow"
                    type="reset"
                    data-rcl-form-action="reset"
                    disabled={submitting}
                  >
                    {libraryStrings("patterns.async-form-flow.reset", "Reset")}
                  </button>
                  <button
                    data-testid="patterns.async-form-flow"
                    type="submit"
                    data-rcl-form-action="submit"
                    disabled={submitting}
                    aria-busy={submitting || undefined}
                  >
                    {submitting ? "Saving…" : submitLabel}
                  </button>
                </FormActions>
              </Form>
            )}
          </div>
        </AsyncBoundary>
      </section>
    </>
  );
});
