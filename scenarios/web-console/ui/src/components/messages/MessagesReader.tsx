import { useDeferredValue, useEffect, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AArrowDown, AArrowUp, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Copy, MoreHorizontal, Play, Search } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { strings } from "../../consts/strings";
import { MarkdownRenderer } from "../markdown";
import { FIND_HIGHLIGHT_CSS, paintMatches, rangesInElement } from "./findInText";
import { MessageActionList, useMessageActions } from "./MessageActionList";
import type { MessageActionContext } from "./messageActions";
import { speakerKey, timeLabel } from "./speaker";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-reader

/** The reader's text size range and step, in px. */
const READER_FONT_MIN = 12;
const READER_FONT_MAX = 32;
const READER_FONT_STEP = 2;

/** A reader text size, rounded and held to the reader's range. */
export function clampReaderFont(size: number): number {
  return Math.min(READER_FONT_MAX, Math.max(READER_FONT_MIN, Math.round(size)));
}

function touchDistance(touches: TouchList): number {
  const first = touches[0];
  const second = touches[1];
  return first && second ? Math.hypot(first.clientX - second.clientX, first.clientY - second.clientY) : 0;
}

interface MessagesReaderProps {
  /** The message's actions as its row offers them, the reader itself excluded. */
  actionContext: MessageActionContext;
  fontSize: number;
  onFontSizeChange: (size: number) => void;
  coarsePointer: boolean;
  onClose: () => void;
  /** Absent on read-only surfaces. */
  onPlay?: (eventId: string) => void;
  /** Open the neighbouring reply; absent at either end. */
  onPrev?: () => void;
  onNext?: () => void;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
}

/**
 * A long reply in full: its own scroll, find-in-message, copy, play, the
 * message's full action list, stepping between replies, and a text size, over
 * the list, which stays mounted and unmoved underneath.
 */
export function MessagesReader({
  actionContext,
  fontSize,
  onFontSizeChange,
  coarsePointer,
  onClose,
  onPlay,
  onPrev,
  onNext,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MessagesReaderProps) {
  const { t, i18n } = useTranslation();
  const { event, isPlaintext } = actionContext;
  const bodyRef = useRef<HTMLDivElement | null>(null);
  // The body mounts inside the drawer's portal a commit after the reader, so
  // the pinch listeners follow the node rather than the first render.
  const [bodyEl, setBodyEl] = useState<HTMLDivElement | null>(null);
  const findRef = useRef<HTMLInputElement | null>(null);
  const moreRef = useRef<HTMLSpanElement | null>(null);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const [matchCount, setMatchCount] = useState(0);
  const [activeIndex, setActiveIndex] = useState(0);
  const [actionsOpen, setActionsOpen] = useState(false);
  const [pinchSize, setPinchSize] = useState<number | null>(null);
  const { ctx, actions, composites } = useMessageActions(actionContext, moreRef, false);

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

  // Another reply starts at its top. The body's parent is the drawer's scroll region.
  useLayoutEffect(() => {
    const scroller = bodyRef.current?.parentElement;
    if (scroller) scroller.scrollTop = 0;
  }, [event.id]);

  // Pinch: two fingers scale the text, shown live and kept when they lift.
  // `touch-action` keeps the page's own pinch-zoom off the text so the gesture
  // arrives here; iOS's gesture events are cancelled for the same reason.
  const fontSizeRef = useRef(fontSize);
  fontSizeRef.current = fontSize;
  const onFontSizeChangeRef = useRef(onFontSizeChange);
  onFontSizeChangeRef.current = onFontSizeChange;
  useEffect(() => {
    if (!bodyEl) return;
    let pinch: { distance: number; size: number; latest: number | null } | null = null;
    const onStart = (touchEvent: TouchEvent) => {
      if (touchEvent.touches.length === 2) pinch = { distance: touchDistance(touchEvent.touches), size: fontSizeRef.current, latest: null };
    };
    const onMove = (touchEvent: TouchEvent) => {
      if (!pinch || touchEvent.touches.length !== 2 || pinch.distance <= 0) return;
      touchEvent.preventDefault();
      pinch.latest = clampReaderFont(pinch.size * touchDistance(touchEvent.touches) / pinch.distance);
      setPinchSize(pinch.latest);
    };
    const onEnd = (touchEvent: TouchEvent) => {
      if (!pinch || touchEvent.touches.length >= 2) return;
      if (pinch.latest != null) onFontSizeChangeRef.current(pinch.latest);
      pinch = null;
      setPinchSize(null);
    };
    const cancelGesture = (gestureEvent: Event) => { gestureEvent.preventDefault(); };
    bodyEl.addEventListener("touchstart", onStart, { passive: true });
    bodyEl.addEventListener("touchmove", onMove, { passive: false });
    bodyEl.addEventListener("touchend", onEnd);
    bodyEl.addEventListener("touchcancel", onEnd);
    bodyEl.addEventListener("gesturestart", cancelGesture);
    return () => {
      bodyEl.removeEventListener("touchstart", onStart);
      bodyEl.removeEventListener("touchmove", onMove);
      bodyEl.removeEventListener("touchend", onEnd);
      bodyEl.removeEventListener("touchcancel", onEnd);
      bodyEl.removeEventListener("gesturestart", cancelGesture);
    };
  }, [bodyEl]);

  const step = (delta: number) => {
    if (matchCount === 0) return;
    setActiveIndex((index) => (index + delta + matchCount) % matchCount);
  };
  const hasQuery = query.trim() !== "";

  return (
    <FullPageDrawer
      avoidKeyboard
      open
      onOpenChange={(open) => { if (!open) onClose(); }}
      title={t(speakerKey(event) as never)}
      closeLabel={t(strings.reader.close)}
      testId="messages-reader"
      initialFocusRef={findRef}
      contentPadding="comfortable"
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
            onClick={() => { ctx.onCopy(event.id, event.text); }}
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
          <span ref={moreRef} className="inline-flex">
            <IconButton
              data-testid="reader-more"
              aria-label={t(strings.messageActions.more)}
              aria-haspopup="menu"
              surface="soft"
              size="xs"
              onClick={() => { setActionsOpen(true); }}
            >
              <MoreHorizontal />
            </IconButton>
          </span>
        </>
      )}
      subheader={(
        <div style={{ paddingInline: "var(--space-md)", paddingBlock: "var(--space-sm)" }}>
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
      )}
      footer={(
        <div className="flex w-full items-center justify-between gap-2">
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
              disabled={fontSize <= READER_FONT_MIN}
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
              disabled={fontSize >= READER_FONT_MAX}
              onClick={() => { onFontSizeChange(clampReaderFont(fontSize + READER_FONT_STEP)); }}
            >
              <AArrowUp />
            </IconButton>
          </div>
          <IconButton data-testid="reader-next-reply" aria-label={t(strings.reader.nextReply)} title={t(strings.reader.nextReply)} surface="soft" size="sm" disabled={!onNext} onClick={onNext}>
            <ChevronRight />
          </IconButton>
        </div>
      )}
    >
      <style>{FIND_HIGHLIGHT_CSS}</style>
      <div
        ref={(node) => { bodyRef.current = node; setBodyEl(node); }}
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
    </FullPageDrawer>
  );
}
