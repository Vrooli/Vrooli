import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { CornerDownLeft, Terminal } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { strings } from "../../consts/strings";
import { normalizeSentText, type SentEcho } from "../../lib/echoMatch";
import { timeLabel } from "./speaker";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

/** First look at the terminal after a send, then how often to look again. */
export const ECHO_FIRST_CHECK_MS = 1000;
export const ECHO_RECHECK_MS = 5000;
/** With no sign of the send by now, say so. */
export const ECHO_NOT_SEEN_MS = 60_000;
/** An unmatched echo leaves Messages after this. */
export const ECHO_EXPIRE_MS = 5 * 60_000;

type Delivery = "sending" | "unsubmitted" | "not-seen";

const LABEL: Readonly<Record<Delivery, string>> = {
  sending: strings.messagesPane.echo.sending,
  unsubmitted: strings.messagesPane.echo.unsubmitted,
  "not-seen": strings.messagesPane.echo.notSeen,
};

/**
 * The terminal soft-wraps a long line at its width, mid-word or at a space the
 * screen text then drops, so the send and the screen are compared without
 * whitespace.
 */
function shownOnScreen(screen: string | null, text: string): boolean {
  const wanted = normalizeSentText(text).replace(/\s/gu, "");
  return screen !== null && wanted !== "" && screen.replace(/\s/gu, "").includes(wanted);
}

interface EchoRowProps {
  echo: SentEcho;
  /** The session's current terminal screen as text (the server's), or null when unavailable. */
  getTerminalText?: () => Promise<string | null>;
  onPressEnter?: () => void;
  onOpenTerminal?: () => void;
  onExpire: (echoId: string) => void;
}

/**
 * A send waiting for the harness to record it: a dimmed user row saying what
 * happened to it. When the text sits on the terminal screen unsubmitted, it
 * offers Enter and the terminal; with no trace after a minute it says so. The
 * checks are component-local and bounded: they stop when the echo resolves
 * (the real row replaces it) or expires.
 */
export function EchoRow({ echo, getTerminalText, onPressEnter, onOpenTerminal, onExpire }: EchoRowProps) {
  const { t } = useTranslation();
  const [delivery, setDelivery] = useState<Delivery>("sending");

  useEffect(() => {
    let settled = false;
    let cancelled = false;
    let recheck: ReturnType<typeof setInterval> | undefined;
    const stopChecking = () => {
      settled = true;
      clearInterval(recheck);
    };
    // Either can flip while a screen read is in flight.
    const stale = () => cancelled || settled;
    // Reads the screen 1 s after the send and every 5 s after that, and stops
    // for good once the answer is known: on screen, or not seen within 60 s.
    const check = async () => {
      if (stale()) return;
      const screen = getTerminalText ? await getTerminalText().catch(() => null) : null;
      if (stale()) return;
      if (shownOnScreen(screen, echo.text)) {
        stopChecking();
        setDelivery("unsubmitted");
      } else if (Date.now() - echo.sentAt >= ECHO_NOT_SEEN_MS) {
        stopChecking();
        setDelivery("not-seen");
      }
    };
    const after = (ms: number) => Math.max(0, echo.sentAt + ms - Date.now());
    const first = setTimeout(() => {
      void check();
      if (!settled) recheck = setInterval(() => { void check(); }, ECHO_RECHECK_MS);
    }, after(ECHO_FIRST_CHECK_MS));
    const notSeen = setTimeout(() => { void check(); }, after(ECHO_NOT_SEEN_MS));
    const expire = setTimeout(() => { onExpire(echo.id); }, after(ECHO_EXPIRE_MS));
    return () => {
      cancelled = true;
      clearTimeout(first);
      clearTimeout(notSeen);
      clearTimeout(expire);
      clearInterval(recheck);
    };
  }, [echo.id, echo.sentAt, echo.text, getTerminalText, onExpire]);

  const offerEnter = delivery === "unsubmitted" && onPressEnter;
  const offerTerminal = delivery !== "sending" && onOpenTerminal;

  return (
    <div
      data-testid="msg-echo-row"
      data-state={delivery}
      aria-label={t(strings.messagesPane.echo.label)}
      className="mx-3 mb-2 rounded-md border border-dashed border-wc-default px-3 py-2 [overflow-anchor:none]"
    >
      <div className="flex flex-wrap items-center gap-x-2 text-xs text-wc-text-secondary opacity-80">
        <span className="font-medium">{t(strings.messagesPane.speaker.you)}</span>
        <span>{timeLabel(new Date(echo.sentAt).toISOString())}</span>
        <span aria-live="polite">· {t(LABEL[delivery] as never)}</span>
      </div>
      <p className="mt-1 whitespace-pre-wrap break-words text-sm text-wc-text-primary opacity-70">{echo.text}</p>
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
