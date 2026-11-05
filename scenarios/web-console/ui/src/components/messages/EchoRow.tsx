import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { CornerDownLeft, Terminal } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { strings } from "../../consts/strings";
import type { OutboxState, SentEcho } from "../../lib/echoMatch";
import { timeLabel } from "./speaker";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

/** An outbox entry the harness never recorded leaves Messages after this. */
export const ECHO_EXPIRE_MS = 5 * 60_000;

const LABEL: Readonly<Record<OutboxState, string>> = {
  staged: strings.messagesPane.echo.staged,
  submitted: strings.messagesPane.echo.submitted,
  queued: strings.messagesPane.echo.queued,
};

interface EchoRowProps {
  echo: SentEcho;
  /** Presses Enter in the session's terminal and marks the send submitted. */
  onPressEnter?: () => void;
  onOpenTerminal?: () => void;
  onExpire: (echoId: string) => void;
}

/**
 * A send waiting for the harness to record it: a dimmed user row saying what
 * happened to it, read from the authoritative outbox state rather than guessed
 * from the terminal screen. A staged send offers Enter (and the terminal); a
 * submitted or queued one just reports where it is. The row clears when the
 * real turn is recorded, or expires on its own.
 */
export function EchoRow({ echo, onPressEnter, onOpenTerminal, onExpire }: EchoRowProps) {
  const { t } = useTranslation();

  useEffect(() => {
    const remaining = Math.max(0, echo.sentAt + ECHO_EXPIRE_MS - Date.now());
    const expire = setTimeout(() => { onExpire(echo.id); }, remaining);
    return () => { clearTimeout(expire); };
  }, [echo.id, echo.sentAt, onExpire]);

  const offerEnter = echo.state === "staged" && Boolean(onPressEnter);
  const offerTerminal = Boolean(onOpenTerminal);

  return (
    <div
      data-testid="msg-echo-row"
      data-state={echo.state}
      aria-label={t(strings.messagesPane.echo.label)}
      className="mx-3 mb-2 rounded-md border border-dashed border-wc-default px-3 py-2 [overflow-anchor:none]"
    >
      <div className="flex flex-wrap items-center gap-x-2 text-xs text-wc-text-secondary opacity-80">
        <span className="font-medium">{t(strings.messagesPane.speaker.you)}</span>
        <span>{timeLabel(new Date(echo.sentAt).toISOString())}</span>
        <span aria-live="polite">· {t(LABEL[echo.state] as never)}</span>
        {echo.attachmentCount > 0 && (
          <span data-testid="msg-echo-attachments">· {t(strings.messagesPane.echo.attachments, { count: echo.attachmentCount })}</span>
        )}
      </div>
      {echo.text !== "" && (
        <p className="mt-1 whitespace-pre-wrap break-words text-sm text-wc-text-primary opacity-70">{echo.text}</p>
      )}
      {(offerEnter || offerTerminal) && (
        <div className="mt-2 flex flex-wrap gap-2">
          {offerEnter && (
            <Button type="button" size="sm" variant="secondary" data-testid="msg-echo-enter" onClick={onPressEnter} className="min-h-11 md:min-h-8">
              <CornerDownLeft aria-hidden className="me-1.5 h-3.5 w-3.5" />
              {t(strings.messagesPane.echo.pressEnter)}
            </Button>
          )}
          {offerTerminal && (
            <Button type="button" size="sm" variant="ghost" data-testid="msg-echo-open-terminal" onClick={onOpenTerminal} className="min-h-11 md:min-h-8">
              <Terminal aria-hidden className="me-1.5 h-3.5 w-3.5" />
              {t(strings.messagesPane.echo.openTerminal)}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
