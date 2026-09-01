/**
 * @libraryId react-component-library:ErrorState
 * @displayName ErrorState
 * @description The recoverable failure composition with a plain-language explanation, a diagnostic-detail policy, retry, alternative actions, and support context that leaks nothing sensitive.
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ErrorState */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/**
 * 1.1.0 — somewhere for the cause to go.
 *
 * The props were `title`, `message` and `onRetry`, which left an adopting app
 * exactly one place to put everything it knew about a failure. What that
 * produced in the wild was a transport dump rendered as body copy:
 *
 *   node reach transport node="25c7e426-…" verb="vrooli-onboarding" scenario
 *   proxy returned 502 Bad Gateway: scenario method api/v2.operator-inputs is
 *   not in the governed catalog
 *
 * — six lines of internal vocabulary in the position where the reader is
 * looking for a sentence, and no room left for the one action that would fix
 * it. The string is not the problem; its altitude is. An operator needs to
 * know what broke and what to do, and an engineer needs the identifiers
 * verbatim, and those are two different readers of the same surface.
 *
 * `detail` is therefore rendered collapsed, in a monospace block that
 * preserves whitespace, under a summary the operator can ignore. `message`
 * goes back to being a sentence. `correlationId` is separate from `detail`
 * because it is the one identifier worth showing even when there is no dump
 * to show, and the one a support conversation always asks for.
 *
 * The disclosure is a native `<details>` on purpose: it is keyboard-operable,
 * findable by in-page search once opened, and it survives with no JavaScript —
 * all three of which matter most on a surface that only appears when something
 * else has already gone wrong.
 */

export interface ErrorStateProps {
  title?: ReactNode;
  message?: ReactNode;
  onRetry?: () => void | Promise<void>;
  /**
   * The verbatim cause: a stack, a transport error, a server payload. Rendered
   * collapsed and monospaced. Never put the operator's explanation here.
   */
  detail?: ReactNode;
  detailLabel?: ReactNode;
  /** Shown with the detail, and on its own when there is no detail. */
  correlationId?: string;
  /** Actions beyond retry — "Re-apply profile", "Open logs". */
  actions?: ReactNode;
  children?: ReactNode;
}

const styles = `
[data-rcl-error-detail] { display: grid; gap: var(--space-2xs, 8px); margin-block-start: var(--space-2xs, 8px); min-inline-size: 0; }
[data-rcl-error-disclosure] { min-inline-size: 0; }
[data-rcl-error-disclosure] > summary {
  cursor: pointer; list-style: none; display: inline-flex; align-items: center; gap: var(--space-3xs, 4px);
  color: var(--color-muted-foreground); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans));
  border-radius: var(--radius-control, .375rem);
}
[data-rcl-error-disclosure] > summary::-webkit-details-marker { display: none; }
[data-rcl-error-disclosure] > summary::after { content: "▾"; font-size: .8em; transition: transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2,0,0,1)); }
[data-rcl-error-disclosure][open] > summary::after { transform: rotate(180deg); }
[data-rcl-error-disclosure] > summary:hover { color: var(--color-foreground); }
[data-rcl-error-disclosure] > summary:focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus); outline-offset: var(--space-3xs, 4px); }
[data-rcl-error-trace] {
  margin: var(--space-2xs, 8px) 0 0;
  padding: var(--space-2xs, 8px);
  max-block-size: 14rem; overflow: auto;
  border: var(--border-hairline, 1px) solid var(--color-border);
  border-radius: var(--radius-control, .375rem);
  background: var(--color-surface-muted, var(--color-surface));
  color: var(--color-muted-foreground);
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: var(--text-caption-size, .75rem); line-height: 1.5;
  white-space: pre-wrap; overflow-wrap: anywhere;
}
[data-rcl-error-correlation] {
  color: var(--color-muted-foreground);
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: var(--text-caption-size, .75rem);
  user-select: all;
}
[data-rcl-error-actions] { display: flex; flex-wrap: wrap; gap: var(--space-2xs, 8px); margin-block-start: var(--space-2xs, 8px); }
@media (prefers-reduced-motion: reduce) { [data-rcl-error-disclosure] > summary::after { transition: none; } }
`;

export const ErrorState = withClassName(function ErrorState({
  title,
  message = "The operation could not be completed.",
  onRetry,
  detail,
  detailLabel,
  correlationId,
  actions,
  children,
}: ErrorStateProps) {
  const libraryStrings = useStrings();
  const resolvedTitle =
    title ?? libraryStrings("feedback.error-state.something-went-wrong", "Something went wrong");
  const resolvedDetailLabel =
    detailLabel ?? libraryStrings("feedback.error-state.technical-detail", "Technical detail");

  // Composed into `error` rather than passed as children: the boundary owns
  // whether children survive an error status, and the cause must appear
  // whatever it decides.
  const explanation = (
    <>
      <StyleSheet name="error-state-1-1" css={styles} />
      {message}
      {(detail || correlationId || actions) && (
        <div data-rcl-error-detail>
          {detail && (
            <details data-rcl-error-disclosure data-testid="feedback.error-state-disclosure">
              <summary>{resolvedDetailLabel}</summary>
              <pre data-rcl-error-trace data-testid="feedback.error-state-trace">
                {detail}
              </pre>
              {correlationId && (
                <p data-rcl-error-correlation data-testid="feedback.error-state-correlation">
                  {correlationId}
                </p>
              )}
            </details>
          )}
          {!detail && correlationId && (
            <p data-rcl-error-correlation data-testid="feedback.error-state-correlation">
              {correlationId}
            </p>
          )}
          {actions && <div data-rcl-error-actions>{actions}</div>}
        </div>
      )}
    </>
  );

  return (
    <AsyncBoundary
      data-testid="feedback.error-state"
      status="error"
      errorTitle={resolvedTitle}
      error={explanation}
      retry={onRetry}
    >
      {children}
    </AsyncBoundary>
  );
});

export default ErrorState;
