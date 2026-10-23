import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";

interface ReturnToLivePillProps {
  /** Whether the pane is scrolled back / parked in backend copy-mode. */
  visible: boolean;
  /** Cancel the stranded copy-mode and return the pane to the live program. */
  onReturnToLive: () => void;
}

/**
 * Persistent recovery control shown while a pane is scrolled back into backend
 * copy-mode. Unlike the transient ScrollHistoryHint, this stays up until the
 * pane returns to live, because it is the escape hatch for exactly the state
 * where typing can no longer reach the shell (a follower with no input lease,
 * or a mobile gesture layer wedged mid-scroll). It is a plain button, so it
 * works even when the terminal's gesture/keystroke path is confused — the tap
 * sends an idempotent `return_to_live` frame straight over the WebSocket.
 */
export function ReturnToLivePill({ visible, onReturnToLive }: ReturnToLivePillProps) {
  const { t } = useTranslation();
  if (!visible) return null;
  return (
    <div
      data-testid="return-to-live-pill"
      role="status"
      // Sits just above the tmux status line (bottom two rows), centered, so it
      // never covers the status line and reads as a compact recovery affordance.
      className="pointer-events-none absolute bottom-12 left-1/2 z-wc-chrome-raised flex max-w-[94%] -translate-x-1/2 items-center gap-1.5 whitespace-nowrap rounded-full bg-slate-900/95 px-3 py-1 text-xs text-slate-200 shadow-lg ring-1 ring-white/10"
    >
      <span aria-hidden className="text-amber-400">↑</span>
      <span>{t(strings.terminalPane.returnToLiveLabel)}</span>
      <span aria-hidden className="text-slate-500">·</span>
      <button
        type="button"
        data-testid="return-to-live-action"
        className="pointer-events-auto font-medium text-sky-400 hover:text-sky-300"
        onClick={onReturnToLive}
      >
        {t(strings.terminalPane.returnToLiveAction)}
      </button>
    </div>
  );
}

export default ReturnToLivePill;
