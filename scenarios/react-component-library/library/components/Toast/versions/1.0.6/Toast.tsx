/**
 * @libraryId react-component-library:Toast
 * @displayName Toast
 * @description
 * @version 1.0.6
 * @tags ["feedback","async","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource feedback.toast */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { CSSProperties } from "react";
import { Presence } from "@vrooli/react-component-library/Presence/1.0.0";
import { Surface } from "@vrooli/react-component-library/Surface/1.0.0";
import {
  useToastManager,
  type ToastRecord,
  type ToastTone,
} from "@vrooli/react-component-library/ToastManager/1.0.0";

const styles = `
[data-rcl-toast-viewport] { position: fixed; z-index: var(--layer-toast, 1000); inset-inline: auto var(--space-lg); inset-block: auto calc(var(--space-lg) + env(safe-area-inset-bottom)); display: grid; gap: var(--space-xs); inline-size: min(100% - (var(--space-lg) * 2), 26rem); pointer-events: none; }
[data-rcl-toast] { pointer-events: auto; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: start; gap: var(--space-sm); padding: var(--space-md); border-inline-start: var(--border-strong) solid currentColor; }
[data-rcl-toast][data-tone="info"] { color: var(--color-info); }
[data-rcl-toast][data-tone="success"] { color: var(--color-success); }
[data-rcl-toast][data-tone="warning"] { color: var(--color-warning); }
[data-rcl-toast][data-tone="error"] { color: var(--color-danger); }
[data-rcl-toast-icon] { display: grid; place-items: center; inline-size: var(--space-lg); block-size: var(--space-lg); border: var(--border-hairline) solid currentColor; border-radius: var(--radius-pill); font: var(--text-label); }
[data-rcl-toast-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-toast-title] { font: var(--text-label); }
[data-rcl-toast-message] { color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-toast-close], [data-rcl-toast-action] { min-block-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); cursor: pointer; font: var(--text-label); }
[data-rcl-toast-close] { inline-size: var(--tap-target-min); font-size: var(--text-title-size); }
[data-rcl-toast-close]:hover, [data-rcl-toast-action]:hover { background: var(--color-surface-muted); }
[data-rcl-toast-close]:focus-visible, [data-rcl-toast-action]:focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (max-width: 38rem) { [data-rcl-toast-viewport] { inset-inline: var(--space-sm); inline-size: auto; } [data-rcl-toast] { padding: var(--space-sm); } }
`;

const toneGlyph: Record<ToastTone, string> = {
  info: "i",
  success: "✓",
  warning: "!",
  error: "!",
};

export interface ToastProps {
  className?: string;
  style?: CSSProperties;
  label?: string;
}

function ToastItem({ toast, onDismiss }: { toast: ToastRecord; onDismiss: () => void }) {
  const strings = useStrings();
  return (
    <Presence present duration="quick">
      <Surface
        elevation="floating"
        data-rcl-toast
        data-tone={toast.tone}
        role={toast.tone === "error" ? "alert" : "status"}
        aria-live={toast.tone === "error" ? "assertive" : "polite"}
        aria-atomic="true"
      >
        <span data-rcl-toast-icon aria-hidden="true">
          {toneGlyph[toast.tone]}
        </span>
        <div data-rcl-toast-copy>
          <span data-rcl-toast-title>{toast.title}</span>
          {toast.message && <span data-rcl-toast-message>{toast.message}</span>}
          {toast.action && (
            <button
              data-testid="feedback.toast"
              data-rcl-toast-action
              type="button"
              onClick={toast.action.onSelect}
            >
              {toast.action.label}
            </button>
          )}
        </div>
        {toast.dismissible !== false && (
          <button
            data-testid="feedback.toast"
            data-rcl-toast-close
            type="button"
            aria-label={strings("feedback.toast.dismiss-notification", "Dismiss notification")}
            onClick={onDismiss}
          >
            ×
          </button>
        )}
      </Surface>
    </Presence>
  );
}

export const Toast = withClassName(function Toast({ className, label, style }: ToastProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("feedback.toast.notifications", "Notifications");
  const manager = useToastManager();
  return (
    <>
      <style data-rcl-toast-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <div className={className} style={style} data-rcl-toast-viewport aria-label={label}>
        {manager.toasts.map((toast) => (
          <ToastItem key={toast.id} toast={toast} onDismiss={() => manager.dismiss(toast.id)} />
        ))}
      </div>
    </>
  );
});
