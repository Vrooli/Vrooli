import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";

interface ScrollHistoryHintProps {
  /** Whether the hint is currently shown. */
  visible: boolean;
  /** When true, offer a shortcut to the pane's Messages view, which does hold
   *  the agent transcript the terminal cannot scroll back to. */
  canViewMessages: boolean;
  onViewMessages: () => void;
}

/**
 * Transient hint shown when a scroll-up gesture reaches a full-screen app that
 * keeps no terminal scrollback (a Codex pane, vim, etc.). Without it the
 * gesture dies silently and reads as a bug. When the pane has a Messages view,
 * the hint points there — that projection holds the conversation history the
 * terminal genuinely cannot reveal.
 */
export function ScrollHistoryHint({ visible, canViewMessages, onViewMessages }: ScrollHistoryHintProps) {
  const { t } = useTranslation();
  if (!visible) return null;
  return (
    <div
      data-testid="scroll-history-hint"
      role="status"
      // Sits above the tmux status line (bottom two rows) so it never covers
      // it, and stays a single compact line so it reads as a quiet toast
      // rather than a block of text.
      className="pointer-events-none absolute bottom-12 left-1/2 z-wc-chrome-raised flex max-w-[94%] -translate-x-1/2 items-center gap-1.5 whitespace-nowrap rounded-full bg-slate-900/95 px-3 py-1 text-xs text-slate-200 shadow-lg ring-1 ring-white/10"
    >
      <span>{t(strings.terminalPane.noScrollbackHint)}</span>
      {canViewMessages && (
        <>
          <span aria-hidden className="text-slate-500">·</span>
          <button
            type="button"
            className="pointer-events-auto font-medium text-sky-400 hover:text-sky-300"
            onClick={onViewMessages}
          >
            {t(strings.terminalPane.noScrollbackViewMessages)}
          </button>
        </>
      )}
    </div>
  );
}

export default ScrollHistoryHint;
