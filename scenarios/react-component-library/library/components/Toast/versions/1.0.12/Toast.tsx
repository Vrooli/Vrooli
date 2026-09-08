/**
 * @libraryId react-component-library:Toast
 * @displayName Toast
 * @description The transient notification rendered through the shared toast service, with queueing, deduplication, in-place updates, action affordances, timeout policy, and announcement.
 * @version 1.0.12
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource feedback.toast */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";
import { MorphingIcon } from "@vrooli/react-component-library/MorphingIcon/3";
import { Presence } from "@vrooli/react-component-library/Presence/1";
import { Surface } from "@vrooli/react-component-library/Surface/1";
import { useSwipeGesture } from "@vrooli/react-component-library/useSwipeGesture/2";
import {
  useToastManager,
  type ToastRecord,
  type ToastTone,
} from "@vrooli/react-component-library/ToastManager/1";

const styles = `
[data-rcl-toast-viewport] { position: fixed; z-index: var(--layer-toast, 500); inset-inline: auto var(--space-lg); inset-block: auto calc(var(--space-lg) + env(safe-area-inset-bottom)); display: grid; gap: var(--space-xs); inline-size: min(100% - (var(--space-lg) * 2), 26rem); pointer-events: none; }
[data-rcl-toast] { pointer-events: auto; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: start; gap: var(--space-sm); padding: var(--space-md); border-inline-start: var(--border-strong) solid currentColor; overflow: hidden; touch-action: pan-y; transition: transform var(--dur-quick) var(--ease-standard); }
[data-rcl-toast] * { touch-action: pan-y; }
[data-rcl-toast][data-dragging="true"] { transition: none; }
[data-rcl-toast][data-tone="info"] { color: var(--color-info); }
[data-rcl-toast][data-tone="success"] { color: var(--color-success); }
[data-rcl-toast][data-tone="warning"] { color: var(--color-warning); }
[data-rcl-toast][data-tone="error"] { color: var(--color-danger); }

[data-rcl-toast-icon] { display: grid; place-items: center; inline-size: var(--icon-size-md, 20px); block-size: var(--icon-size-md, 20px); margin-block-start: 1px; color: currentColor; }
[data-rcl-toast-icon] svg { inline-size: 100%; block-size: 100%; display: block; }

/* Every line has to survive text it did not choose: a URL, a filesystem path, a stack
 * frame. Without this, one unbroken token renders past the viewport's inline size and
 * the notice is unreadable exactly when it carries an error. */
[data-rcl-toast-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-toast-title] { font: var(--text-label); overflow-wrap: anywhere; }
[data-rcl-toast-message] { color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; white-space: pre-wrap; max-block-size: 8lh; overflow-y: auto; overscroll-behavior: contain; }

[data-rcl-toast-action] { justify-self: start; min-block-size: var(--tap-target-min); padding-inline: var(--space-2xs); border: var(--border-hairline) solid color-mix(in srgb, currentColor 45%, transparent); border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); cursor: pointer; font: var(--text-label); }
[data-rcl-toast-action]:hover { background: color-mix(in srgb, currentColor 18%, transparent); }

/* Dismiss is the least important control in the notice and reads as clutter when boxed
 * like an action. A real tap target would also set the height of the whole row, so the
 * conforming hit area is an ::after overlay at zero layout cost. */
[data-rcl-toast-close] { position: relative; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; padding: var(--space-3xs); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; transition: background-color var(--dur-quick, 180ms) ease, color var(--dur-quick, 180ms) ease; }
[data-rcl-toast-close]::after { content: ""; position: absolute; inset-block-start: 50%; inset-inline-start: 50%; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); transform: translate(-50%, -50%); }
[data-rcl-toast-close]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-toast-close] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); display: block; }

[data-rcl-toast-action]:focus-visible, [data-rcl-toast-close]:focus-visible { outline: var(--focus-ring-width, 2px) solid var(--color-focus-ring, var(--color-focus)); outline-offset: 1px; }

/* Tone must survive a mode that discards author colour, so the glyph carries the same
 * distinction the palette does. */
@media (forced-colors: active) {
  [data-rcl-toast] { border: 1px solid CanvasText; background: Canvas; color: CanvasText; }
  [data-rcl-toast-icon], [data-rcl-toast-close] { color: CanvasText; }
  [data-rcl-toast-action] { border-color: CanvasText; }
}
@media (prefers-reduced-motion: reduce) {
  [data-rcl-toast-close] { transition: none; }
}
@media (max-width: 38rem) { [data-rcl-toast-viewport] { inset-inline: var(--space-sm); inline-size: auto; } [data-rcl-toast] { padding: var(--space-sm); } }
`;

const SWIPE_DISMISS_THRESHOLD = 96;

function ToneIcon({ tone }: { tone: ToastTone }) {
  // Keep the icon instance mounted while the manager patches the same toast. The
  // primitive then owns the transition from send/check/close instead of the toast
  // replaying its enter animation on every progress update.
  if (tone === "warning") {
    return (
      <MorphingIcon iconKey="warning" morph="auto" size="md">
        <svg
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.75"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M10 2.5L18.5 17.5H1.5z" />
          <path d="M10 8v3.5" />
          <path d="M10 14.5h.01" />
        </svg>
      </MorphingIcon>
    );
  }
  const icon = tone === "success" ? "check" : tone === "error" ? "close" : "send";
  return <MorphingIcon icon={icon} morph="auto" size="md" />;
}

function DismissIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <path d="M4 4l8 8" />
      <path d="M12 4l-8 8" />
    </svg>
  );
}

export interface ToastProps {
  className?: string;
  style?: CSSProperties;
  label?: string;
}

/** One toast on screen, plus whether it is on its way out. */
interface RenderedToast {
  record: ToastRecord;
  present: boolean;
}

/**
 * The manager owns when a toast stops being true; this component owns how long it stays
 * on screen after that. Rendering `manager.toasts` directly means a dismissed toast is
 * unmounted the instant it leaves the array, so its exit animation never runs — the
 * notice does not leave, it vanishes. Departing records are held here, marked absent, and
 * dropped only when the animation reports it has finished.
 */
function useRenderedToasts(toasts: ToastRecord[]): [RenderedToast[], (id: string) => void] {
  const [rendered, setRendered] = useState<RenderedToast[]>(() =>
    toasts.map((record) => ({ record, present: true })),
  );

  useEffect(() => {
    setRendered((current) => {
      const live = new Map(toasts.map((record) => [record.id, record]));
      // Departing toasts keep their slot so the survivors do not jump upward mid-exit.
      const next: RenderedToast[] = current.map((item) => {
        const record = live.get(item.record.id);
        return record ? { record, present: true } : { record: item.record, present: false };
      });
      const known = new Set(current.map((item) => item.record.id));
      for (const record of toasts) {
        if (!known.has(record.id)) next.push({ record, present: true });
      }
      return next;
    });
  }, [toasts]);

  const forget = useCallback((id: string) => {
    setRendered((current) => current.filter((item) => item.record.id !== id || item.present));
  }, []);

  return [rendered, forget];
}

function ToastItem({
  toast,
  present,
  onDismiss,
  onExitComplete,
}: {
  toast: ToastRecord;
  present: boolean;
  onDismiss: () => void;
  onExitComplete: (id: string) => void;
}) {
  const strings = useStrings();
  const surfaceRef = useRef<HTMLDivElement>(null);
  const handleExitComplete = useCallback(
    () => onExitComplete(toast.id),
    [onExitComplete, toast.id],
  );

  const paint = useCallback((translate: number, dragging: boolean) => {
    const surface = surfaceRef.current;
    if (!surface) return;
    surface.dataset.dragging = dragging ? "true" : "false";
    surface.style.transform = translate === 0 ? "" : `translateX(${String(translate)}px)`;
  }, []);

  const dismissOffscreen = useCallback(() => {
    const surface = surfaceRef.current;
    if (!surface) return;
    const distance = surface.getBoundingClientRect().width + 32;
    surface.dataset.dragging = "false";
    surface.style.transform = `translateX(-${String(distance)}px)`;
  }, []);

  const swipe = useSwipeGesture({
    // The viewport is anchored to the inline end, so a leftward swipe carries a
    // notice away from its resting position while vertical scrolling remains native.
    direction: "left",
    stages: [SWIPE_DISMISS_THRESHOLD],
    releaseMode: "commit",
    onMove: ({ translate }) => paint(translate, true),
    onRelease: (release) => {
      if (release.outcome === "commit") {
        dismissOffscreen();
        onDismiss();
      } else {
        paint(0, false);
      }
    },
  });

  return (
    <Presence present={present} duration="quick" onExitComplete={handleExitComplete}>
      <Surface
        ref={surfaceRef}
        elevation="floating"
        data-testid="feedback.toast"
        data-rcl-toast
        data-tone={toast.tone}
        role={toast.tone === "error" ? "alert" : "status"}
        aria-live={toast.tone === "error" ? "assertive" : "polite"}
        aria-atomic="true"
        onPointerDown={swipe.onPointerDown}
      >
        <span data-rcl-toast-icon aria-hidden="true">
          <ToneIcon tone={toast.tone} />
        </span>
        <div data-rcl-toast-copy>
          <span data-rcl-toast-title>{toast.title}</span>
          {toast.message && <span data-rcl-toast-message>{toast.message}</span>}
          {toast.action && (
            <button
              data-testid="feedback.toast.action"
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
            data-testid="feedback.toast.dismiss"
            data-rcl-toast-close
            type="button"
            aria-label={strings("feedback.toast.dismiss-notification", "Dismiss notification")}
            onClick={onDismiss}
          >
            <DismissIcon />
          </button>
        )}
      </Surface>
    </Presence>
  );
}

/**
 * The single viewport. It is a labelled region rather than a bare div: a name on an
 * element with no role is not exposed to assistive technology, so the previous version
 * carried an aria-label that nothing could read.
 */
export const Toast = withClassName(function Toast({ className, label, style }: ToastProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("feedback.toast.notifications", "Notifications");
  const manager = useToastManager();
  const [rendered, forget] = useRenderedToasts(manager.toasts);

  return (
    <>
      <StyleSheet libraryId="react-component-library:Toast" version="1.0.12-draft.1" css={styles} />
      <div
        className={className}
        style={style}
        data-testid="feedback.toast.viewport"
        data-rcl-toast-viewport
        role="region"
        aria-label={label}
        // A notice the reader is still reading must not expire under them, and a reader
        // reaching for the dismiss control is the clearest possible signal of that.
        onPointerEnter={manager.pause}
        onPointerLeave={manager.resume}
        onFocusCapture={manager.pause}
        onBlurCapture={manager.resume}
      >
        {rendered.map((item) => (
          <ToastItem
            key={item.record.id}
            toast={item.record}
            present={item.present}
            onDismiss={() => manager.dismiss(item.record.id)}
            onExitComplete={forget}
          />
        ))}
      </div>
    </>
  );
});
