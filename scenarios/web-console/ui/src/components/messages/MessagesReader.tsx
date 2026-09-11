import { useDeferredValue, useEffect, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChevronDown, ChevronUp, Copy, Play } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";
import { MarkdownRenderer } from "../markdown";
import { FIND_HIGHLIGHT_CSS, paintMatches, rangesInElement } from "./findInText";
import { speakerKey, timeLabel } from "./speaker";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-reader

interface MessagesReaderProps {
  event: ConversationEvent;
  fontSize: number;
  isPlaintext: boolean;
  onClose: () => void;
  onCopy: (eventId: string, text: string) => void;
  /** Absent on read-only surfaces. */
  onPlay?: (eventId: string) => void;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
}

/**
 * A long reply in full: its own scroll, find-in-message, copy, and play, over
 * the list, which stays mounted and unmoved underneath.
 */
export function MessagesReader({
  event,
  fontSize,
  isPlaintext,
  onClose,
  onCopy,
  onPlay,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MessagesReaderProps) {
  const { t, i18n } = useTranslation();
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const findRef = useRef<HTMLInputElement | null>(null);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const [matchCount, setMatchCount] = useState(0);
  const [activeIndex, setActiveIndex] = useState(0);

  // Count matches in the rendered text whenever the query changes.
  useEffect(() => {
    const body = bodyRef.current;
    const count = body ? rangesInElement(body, deferredQuery).length : 0;
    setMatchCount(count);
    setActiveIndex(0);
  }, [deferredQuery, event.text, isPlaintext]);

  // Paint the matches and bring the active one into view; clear on change.
  useLayoutEffect(() => {
    const body = bodyRef.current;
    if (!body || matchCount === 0) return;
    const painted = paintMatches(rangesInElement(body, deferredQuery), activeIndex);
    painted.activeElement?.scrollIntoView({ block: "center" });
    return painted.clear;
  }, [activeIndex, deferredQuery, matchCount]);

  const step = (delta: number) => {
    if (matchCount === 0) return;
    setActiveIndex((index) => (index + delta + matchCount) % matchCount);
  };

  return (
    <FullPageDrawer
      avoidKeyboard
      open
      onOpenChange={(open) => { if (!open) onClose(); }}
      title={t(speakerKey(event) as never)}
      closeLabel={t(strings.reader.close)}
      testId="messages-reader"
      initialFocusRef={findRef}
      headerExtra={(
        <span className="font-mono text-[11px] text-wc-text-faint">
          {timeLabel(event.createdAt, new Date(), i18n.language)} · #{event.sequence}
        </span>
      )}
      headerActions={(
        <>
          <IconButton
            data-testid="reader-copy"
            aria-label={t(strings.messageActions.copy)}
            surface="soft"
            size="xs"
            onClick={() => { onCopy(event.id, event.text); }}
          >
            <Copy />
          </IconButton>
          {onPlay && (
            <IconButton
              data-testid="reader-play"
              aria-label={t(strings.reader.play)}
              surface="soft"
              size="xs"
              onClick={() => { onPlay(event.id); }}
            >
              <Play />
            </IconButton>
          )}
        </>
      )}
      subheader={(
        <div className="flex items-center gap-2">
          <input
            ref={findRef}
            data-testid="reader-find-input"
            type="search"
            value={query}
            onChange={(changeEvent) => { setQuery(changeEvent.target.value); }}
            onKeyDown={(keyEvent) => {
              if (keyEvent.key !== "Enter") return;
              keyEvent.preventDefault();
              step(keyEvent.shiftKey ? -1 : 1);
            }}
            placeholder={t(strings.reader.find)}
            aria-label={t(strings.reader.find)}
            className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary placeholder:text-wc-text-faint"
          />
          <span
            data-testid="reader-match-count"
            data-current={matchCount > 0 ? activeIndex + 1 : 0}
            data-total={matchCount}
            className="shrink-0 font-mono text-xs text-wc-text-muted"
            aria-live="polite"
          >
            {deferredQuery.trim()
              ? (matchCount > 0 ? t(strings.reader.matchCount, { current: activeIndex + 1, total: matchCount }) : t(strings.reader.noMatches))
              : ""}
          </span>
          <IconButton data-testid="reader-find-prev" aria-label={t(strings.reader.prev)} surface="soft" size="xs" disabled={matchCount === 0} onClick={() => { step(-1); }}>
            <ChevronUp />
          </IconButton>
          <IconButton data-testid="reader-find-next" aria-label={t(strings.reader.next)} surface="soft" size="xs" disabled={matchCount === 0} onClick={() => { step(1); }}>
            <ChevronDown />
          </IconButton>
        </div>
      )}
    >
      <style>{FIND_HIGHLIGHT_CSS}</style>
      <div ref={bodyRef} data-testid="messages-reader-body" style={{ fontSize: `${String(fontSize)}px` }} className="text-wc-text-primary">
        {isPlaintext ? (
          <pre className="whitespace-pre-wrap break-words [overflow-wrap:anywhere] font-mono">{event.text}</pre>
        ) : (
          <MarkdownRenderer content={event.text} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} onMermaidOpen={onMermaidOpen} />
        )}
      </div>
    </FullPageDrawer>
  );
}
