import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SendHorizontal, TextCursorInput, Trash2 } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { strings } from "../../consts/strings";
import type { SentHistoryEntry } from "../../hooks/useCommandHistory";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

type Say = (key: string, options?: Record<string, unknown>) => string;

function agoLabel(at: number, now: number, say: Say): string {
  const seconds = Math.max(0, Math.floor((now - at) / 1000));
  if (seconds < 60) return say(strings.sentHistory.justNow);
  if (seconds < 3600) return say(strings.sentHistory.minutesAgo, { n: Math.floor(seconds / 60) });
  if (seconds < 86_400) return say(strings.sentHistory.hoursAgo, { n: Math.floor(seconds / 3600) });
  return say(strings.sentHistory.daysAgo, { n: Math.floor(seconds / 86_400) });
}

interface SentHistorySheetProps {
  open: boolean;
  /** Oldest first, as the history hook keeps them. */
  entries: readonly SentHistoryEntry[];
  onClose: () => void;
  /** Put the text into the draft at the caret. */
  onInsert: (text: string) => void;
  /** Send the text the way Send does: typed, never followed by Enter. */
  onSend: (text: string) => void;
  onClear: () => void;
}

/**
 * The last sends from this device (localStorage only, never synced), newest
 * first, with Insert and Resend. A responsive dialog: a sheet on a phone, a
 * dialog on a desktop.
 */
export function SentHistorySheet({ open, entries, onClose, onInsert, onSend, onClear }: SentHistorySheetProps) {
  const { t } = useTranslation();
  const say = t as unknown as Say;
  const [filter, setFilter] = useState("");
  const rows = useMemo(() => {
    const query = filter.trim().toLowerCase();
    return [...entries].reverse().filter((entry) => query === "" || entry.text.toLowerCase().includes(query));
  }, [entries, filter]);
  if (!open) return null;
  const now = Date.now();

  return (
    <ResponsiveDialog
      open
      onClose={onClose}
      size="md"
      title={t(strings.sentHistory.title)}
      closeLabel={t(strings.sentHistory.close)}
      testId="sent-history-sheet"
      avoidKeyboard
      contentPadding="none"
      subheader={entries.length > 0 ? (
        <div style={{ paddingInline: "var(--space-md)", paddingBlock: "var(--space-sm)" }}>
          <input
            data-testid="sent-history-filter"
            aria-label={t(strings.sentHistory.filterLabel)}
            placeholder={t(strings.sentHistory.filterPlaceholder)}
            value={filter}
            onChange={(event) => { setFilter(event.target.value); }}
            className="w-full rounded-md border border-wc-default bg-wc-surface-input px-3 py-2 text-base text-wc-text-primary outline-none focus:border-wc-accent"
          />
        </div>
      ) : undefined}
    >
      {entries.length === 0 ? (
        <p data-testid="sent-history-empty" className="px-4 py-6 text-center text-sm text-wc-text-secondary">
          {t(strings.sentHistory.empty)}
        </p>
      ) : (
        <>
          <ul className="divide-y divide-wc-default">
            {rows.map((entry) => (
              <li key={`${String(entry.at)}:${entry.text}`} data-testid="sent-history-row" className="flex flex-col gap-2 px-4 py-3">
                <div className="flex items-baseline justify-between gap-3">
                  <p data-testid="sent-history-text" className="min-w-0 flex-1 whitespace-pre-wrap break-words text-sm text-wc-text-primary">
                    {entry.text}
                  </p>
                  <span className="shrink-0 text-xs text-wc-text-secondary">{agoLabel(entry.at, now, say)}</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    data-testid="sent-history-insert"
                    className="min-h-11 md:min-h-8"
                    onClick={() => {
                      onInsert(entry.text);
                      onClose();
                    }}
                  >
                    <TextCursorInput aria-hidden className="me-1.5 h-3.5 w-3.5" />
                    {t(strings.sentHistory.insert)}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    data-testid="sent-history-resend"
                    className="min-h-11 md:min-h-8"
                    onClick={() => {
                      onSend(entry.text);
                      onClose();
                    }}
                  >
                    <SendHorizontal aria-hidden className="me-1.5 h-3.5 w-3.5" />
                    {t(strings.sentHistory.resend)}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
          <div className="px-4 py-3">
            <Button type="button" size="sm" variant="ghost" data-testid="sent-history-clear" className="min-h-11 md:min-h-8" onClick={onClear}>
              <Trash2 aria-hidden className="me-1.5 h-3.5 w-3.5" />
              {t(strings.sentHistory.clear)}
            </Button>
          </div>
        </>
      )}
    </ResponsiveDialog>
  );
}
