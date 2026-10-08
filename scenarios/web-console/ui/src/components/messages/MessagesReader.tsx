import { useDeferredValue, useEffect, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AArrowDown, AArrowUp, ArrowLeft, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, MoreHorizontal, Search } from "lucide-react";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { strings } from "../../consts/strings";
import { copyText } from "../../lib/clipboard";
import { MarkdownRenderer } from "../markdown";
import { FIND_HIGHLIGHT_CSS, paintMatches, rangesInElement } from "./findInText";
import { MessageActionList, useMessageActions } from "./MessageActionList";
import type { MessageActionContext } from "./messageActions";
import { speakerKey, timeLabel } from "./speaker";
import { clampMessagesFont, useMessagesTypographyPinch } from "../../hooks/useMessagesTypographyPinch";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-reader

/** The reader's text size range and step, in px. */
const READER_FONT_STEP = 2;

/** A reader text size, rounded and held to the reader's range. */
export function clampReaderFont(size: number): number {
  return clampMessagesFont(size);
}

interface MessagesReaderProps {
  /** The message's actions as its row offers them, the reader itself excluded. */
  actionContext: MessageActionContext;
  fontSize: number;
  onFontSizeChange: (size: number) => void;
  coarsePointer: boolean;
  /** The phone keyboard is up: the footer gives its room to the text. */
  hideFooter: boolean;
  onClose: () => void;
  /** Open the neighbouring reply; absent at either end. */
  onPrev?: () => void;
  onNext?: () => void;
  /** Replies that arrived after everything this reader has shown. */
  newReplyCount: number;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
}

/**
 * A reply in full, in place of the list inside the pane, so the composer
 * below stays in reach: its own scroll, find-in-message as the header, copy,
 * the message's full action list, stepping between replies, and a text size.
 * The list stays laid out and live underneath, out of reach until it returns.
 */
export function MessagesReader({
  actionContext,
  fontSize,
  onFontSizeChange,
  coarsePointer,
  hideFooter,
  onClose,
  onPrev,
  onNext,
  newReplyCount,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MessagesReaderProps) {
  const { t, i18n } = useTranslation();
  const { event, isPlaintext } = actionContext;
  const rootRef = useRef<HTMLElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const bodyRef = useRef<HTMLDivElement | null>(null);
  // The pinch listeners follow the body node itself, attached and detached with it.
  const findRef = useRef<HTMLInputElement | null>(null);
  const moreRef = useRef<HTMLSpanElement | null>(null);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const [matchCount, setMatchCount] = useState(0);
  const [activeIndex, setActiveIndex] = useState(0);
  const [actionsOpen, setActionsOpen] = useState(false);
  const { ctx, actions, composites } = useMessageActions(actionContext, moreRef, false);

  // On a keyboard the find field takes focus; on touch the reader itself
  // does, so opening it never raises the phone's keyboard.
  useLayoutEffect(() => {
    if (coarsePointer) rootRef.current?.focus({ preventScroll: true });
    else findRef.current?.focus({ preventScroll: true });
    // Only on open; stepping between replies keeps wherever focus is.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  // Another reply starts at its top.
  useLayoutEffect(() => {
    const scroller = scrollRef.current;
    if (scroller) scroller.scrollTop = 0;
  }, [event.id]);

  const pinchSize = useMessagesTypographyPinch(bodyRef, fontSize, onFontSizeChange);

  const step = (delta: number) => {
    if (matchCount === 0) return;
    setActiveIndex((index) => (index + delta + matchCount) % matchCount);
  };
  const hasQuery = query.trim() !== "";
  const speaker: string = t(speakerKey(event) as never);

  return (
    <section
      ref={rootRef}
      data-testid="messages-reader"
      aria-label={speaker}
      tabIndex={-1}
      onKeyDown={(keyEvent) => {
        // An open action list closes on its own Escape; the reader waits for the next.
        if (keyEvent.key !== "Escape" || keyEvent.defaultPrevented || actionsOpen) return;
        keyEvent.preventDefault();
        onClose();
      }}
      className="absolute inset-0 z-wc-chrome-raised flex flex-col bg-wc-surface-base outline-none"
    >
      <style>{FIND_HIGHLIGHT_CSS}</style>
      <header data-testid="reader-header" className="flex shrink-0 items-center gap-1 border-b border-wc-default px-2 py-1.5">
        <IconButton data-testid="reader-back" aria-label={t(strings.reader.back)} title={t(strings.reader.back)} surface="ghost" size="sm" onClick={onClose}>
          <ArrowLeft />
        </IconButton>
        <div className="min-w-0 flex-1">
          <InputGroup size="md" shape="rounded" testId="reader-find-group">
            <InputGroup.Adornment side="leading">
              <Search aria-hidden />
            </InputGroup.Adornment>
            <InputGroup.Field>
              <Input
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
              />
            </InputGroup.Field>
            {/* Stepping only exists once there is something to step through. */}
            {hasQuery && (
              <>
                <InputGroup.Adornment side="trailing">
                  <span
                    data-testid="reader-match-count"
                    data-current={matchCount > 0 ? activeIndex + 1 : 0}
                    data-total={matchCount}
                    className="font-mono text-xs"
                    aria-live="polite"
                  >
                    {matchCount > 0 ? t(strings.reader.matchCount, { current: activeIndex + 1, total: matchCount }) : t(strings.reader.noMatches)}
                  </span>
                </InputGroup.Adornment>
                <InputGroup.Action>
                  <IconButton data-testid="reader-find-prev" aria-label={t(strings.reader.prev)} surface="ghost" size="sm" disabled={matchCount === 0} onClick={() => { step(-1); }}>
                    <ChevronUp />
                  </IconButton>
                </InputGroup.Action>
                <InputGroup.Action>
                  <IconButton data-testid="reader-find-next" aria-label={t(strings.reader.next)} surface="ghost" size="sm" disabled={matchCount === 0} onClick={() => { step(1); }}>
                    <ChevronDown />
                  </IconButton>
                </InputGroup.Action>
              </>
            )}
          </InputGroup>
        </div>
        <CopyIconButton
          data-testid="reader-copy"
          value={event.text}
          writeText={copyText}
          copied={ctx.copied}
          aria-label={t(strings.messageActions.copy)}
          copiedLabel={t(strings.messageActions.copied)}
          failedLabel={t(strings.messageActions.copyFailed)}
          size="sm"
        />
        <span ref={moreRef} className="inline-flex">
          <IconButton
            data-testid="reader-more"
            aria-label={t(strings.messageActions.more)}
            aria-haspopup="menu"
            surface="ghost"
            size="sm"
            onClick={() => { setActionsOpen(true); }}
          >
            <MoreHorizontal />
          </IconButton>
        </span>
      </header>
      <div ref={scrollRef} data-testid="reader-scroll" className="min-h-0 flex-1 overflow-auto overscroll-contain px-4 py-3 sm:px-6">
        <div className="mx-auto max-w-3xl">
          <p data-testid="reader-meta" className="mb-3 flex flex-wrap items-baseline gap-x-2 text-xs">
            <span className="font-medium text-wc-text-secondary">{speaker}</span>
            <span className="font-mono text-[11px] text-wc-text-faint">
              {timeLabel(event.createdAt, new Date(), i18n.language)} · #{event.sequence}
            </span>
          </p>
          <div
            ref={bodyRef}
            data-testid="messages-reader-body"
            style={{ fontSize: `${String(pinchSize ?? fontSize)}px` }}
            className="touch-pan-y text-wc-text-primary"
          >
            {isPlaintext ? (
              <pre className="whitespace-pre-wrap break-words [overflow-wrap:anywhere] font-mono">{event.text}</pre>
            ) : (
              <MarkdownRenderer content={event.text} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} onMermaidOpen={onMermaidOpen} />
            )}
          </div>
        </div>
      </div>
      {!hideFooter && (
        // Its height is a shared token: the playback pill rises by it while a reader is open.
        <footer
          data-testid="reader-footer"
          style={{ blockSize: "var(--wc-reader-footer-h)" }}
          className="flex shrink-0 items-center justify-between gap-2 border-t border-wc-default px-3"
        >
          <IconButton data-testid="reader-prev-reply" aria-label={t(strings.reader.prevReply)} title={t(strings.reader.prevReply)} surface="soft" size="sm" disabled={!onPrev} onClick={onPrev}>
            <ChevronLeft />
          </IconButton>
          <div className="flex items-center gap-1">
            <IconButton
              data-testid="reader-font-smaller"
              aria-label={t(strings.reader.smaller)}
              title={t(strings.reader.smaller)}
              surface="soft"
              size="sm"
              disabled={fontSize <= 12}
              onClick={() => { onFontSizeChange(clampReaderFont(fontSize - READER_FONT_STEP)); }}
            >
              <AArrowDown />
            </IconButton>
            <IconButton
              data-testid="reader-font-larger"
              aria-label={t(strings.reader.larger)}
              title={t(strings.reader.larger)}
              surface="soft"
              size="sm"
              disabled={fontSize >= 32}
              onClick={() => { onFontSizeChange(clampReaderFont(fontSize + READER_FONT_STEP)); }}
            >
              <AArrowUp />
            </IconButton>
          </div>
          <span className="relative inline-flex">
            <IconButton
              data-testid="reader-next-reply"
              data-new-count={newReplyCount > 0 ? newReplyCount : undefined}
              aria-label={newReplyCount > 0 ? t(strings.reader.nextReplyNew, { count: newReplyCount }) : t(strings.reader.nextReply)}
              title={newReplyCount > 0 ? t(strings.reader.nextReplyNew, { count: newReplyCount }) : t(strings.reader.nextReply)}
              surface={newReplyCount > 0 ? "solid" : "soft"}
              size="sm"
              disabled={!onNext}
              onClick={onNext}
            >
              <ChevronRight />
            </IconButton>
            {newReplyCount > 0 && (
              <span
                aria-hidden
                className="pointer-events-none absolute -end-1 -top-1 min-w-4 rounded-full bg-wc-accent px-1 text-center text-[10px] font-semibold leading-4 text-wc-accent-fg"
              >
                {newReplyCount}
              </span>
            )}
          </span>
        </footer>
      )}
      {actionsOpen && (
        <MessageActionList
          actions={actions}
          ctx={ctx}
          coarsePointer={coarsePointer}
          origin={null}
          anchorRef={moreRef}
          onClose={() => { setActionsOpen(false); }}
        />
      )}
      {composites}
    </section>
  );
}
