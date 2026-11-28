import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Spinner } from "@vrooli/react-component-library/Spinner";
import { MarkdownRenderer } from "@vrooli/file-viewer";
import { getConversationSession, type ConversationEvent, type ConversationThreadSummary } from "../../api/conversation";
import { sessionPreviewSource } from "../../api/filePreview";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { PAGE_SIZE } from "../../lib/conversationLoad";
import { isHumanMessage, isWorkRow, mergeProjectedRows } from "../../lib/conversationRows";
import { NoticeRow } from "./NoticeRow";
import { eventTime, speakerKey, speakerParams } from "./speaker";
import { useTranslate } from "./useTranslate";
import { WorkSpanRow } from "./WorkSpanRow";
import { threadDuration } from "./workSummary";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#agent-threads

interface ThreadViewProps {
  sessionId: string;
  thread: ConversationThreadSummary;
  fontSize: number;
  onClose: () => void;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
}

type ThreadLoad = { state: "loading" } | { state: "failed" } | { state: "loaded"; hasOlder: boolean; oldest: number };

/**
 * One agent thread's own conversation, in the pane in place of the list (as
 * the reader is): its messages with its work collapsed between them, newest
 * at the bottom, refreshed as the thread grows. Back returns to the list
 * exactly where it was.
 */
export function ThreadView({ sessionId, thread, fontSize, onClose, onLinkClick, onFileReferenceClick, onMermaidOpen }: ThreadViewProps) {
  const t = useTranslate();
  const { i18n } = useTranslation();
  const [events, setEvents] = useState<ConversationEvent[]>([]);
  const [load, setLoad] = useState<ThreadLoad>({ state: "loading" });
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
  const scrollRef = useRef<HTMLDivElement>(null);
  const rootRef = useRef<HTMLElement>(null);
  const firstLoad = useRef(true);

  // The newest page, again whenever the thread gains rows.
  useEffect(() => {
    let cancelled = false;
    void getConversationSession(sessionId, { projection: "messages", threadId: thread.threadId, limit: PAGE_SIZE }).then((page) => {
      if (cancelled) return;
      setEvents((current) => mergeProjectedRows(current, page.events).events);
      setLoad((current) => (current.state === "loaded" && !firstLoad.current
        ? current
        : { state: "loaded", hasOlder: page.hasMore ?? false, oldest: page.oldestSequence ?? 0 }));
      if (firstLoad.current) {
        firstLoad.current = false;
        requestAnimationFrame(() => {
          const el = scrollRef.current;
          if (el) el.scrollTop = el.scrollHeight;
        });
      }
    }).catch(() => {
      if (!cancelled && firstLoad.current) setLoad({ state: "failed" });
    });
    return () => { cancelled = true; };
  }, [sessionId, thread.threadId, thread.lastSequence]);

  useEffect(() => { rootRef.current?.focus({ preventScroll: true }); }, []);

  const loadOlder = useCallback(() => {
    if (load.state !== "loaded" || !load.hasOlder) return;
    void getConversationSession(sessionId, { projection: "messages", threadId: thread.threadId, limit: PAGE_SIZE, beforeSequence: load.oldest }).then((page) => {
      setEvents((current) => mergeProjectedRows(current, page.events).events);
      setLoad({ state: "loaded", hasOlder: page.hasMore ?? false, oldest: page.oldestSequence ?? 0 });
    }).catch(() => { /* the button stays; a retry is one tap */ });
  }, [load, sessionId, thread.threadId]);

  const toggleWork = useCallback((eventId: string) => {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(eventId)) next.delete(eventId);
      else next.add(eventId);
      return next;
    });
  }, []);

  const name = thread.label || t(strings.messagesPane.thread.unnamed);
  const duration = threadDuration(thread, t);
  return (
    <section
      ref={rootRef}
      data-testid="thread-view"
      data-thread-id={thread.threadId}
      aria-label={name}
      tabIndex={-1}
      onKeyDown={(keyEvent) => {
        if (keyEvent.key !== "Escape" || keyEvent.defaultPrevented) return;
        keyEvent.preventDefault();
        onClose();
      }}
      className="absolute inset-0 z-wc-chrome-raised flex flex-col bg-wc-surface-base outline-none"
    >
      <header className="flex shrink-0 items-center gap-2 border-b border-wc-default px-2 py-1.5">
        <IconButton data-testid="thread-back" aria-label={t(strings.messagesPane.thread.back)} title={t(strings.messagesPane.thread.back)} surface="ghost" size="sm" onClick={onClose}>
          <ArrowLeft className="rtl:-scale-x-100" />
        </IconButton>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-semibold text-wc-text-primary">{name}</div>
          <div className="truncate text-[11px] text-wc-text-faint">
            {[t(strings.messagesPane.thread.messageCount, { count: thread.messageCount }), ...(duration ? [duration] : []), thread.description].filter(Boolean).join(" · ")}
          </div>
        </div>
      </header>
      <div ref={scrollRef} data-testid="thread-scroll" className="min-h-0 flex-1 overflow-y-auto">
        {load.state === "loading" && (
          <div className="flex justify-center p-6"><Spinner label={t(strings.messagesPane.work.loading)} /></div>
        )}
        {load.state === "failed" && (
          <p role="alert" className="p-6 text-center text-sm text-red-400">{t(strings.messagesPane.thread.loadFailed)}</p>
        )}
        {load.state === "loaded" && load.hasOlder && (
          <div className="flex justify-center p-2">
            <Button type="button" variant="ghost" size="sm" data-testid="thread-load-older" onClick={loadOlder}>
              {t(strings.messagesPane.thread.loadOlder)}
            </Button>
          </div>
        )}
        {load.state === "loaded" && events.length === 0 && (
          <p className="p-6 text-center text-sm text-wc-text-faint">{t(strings.messagesPane.thread.empty)}</p>
        )}
        {events.map((event, index) => {
          if (isWorkRow(event)) {
            return (
              <WorkSpanRow
                key={event.id}
                sessionId={sessionId}
                event={event}
                previousAt={timeOf(events[index - 1])}
                nextAt={timeOf(events[index + 1])}
                expanded={expanded.has(event.id)}
                onToggle={toggleWork}
              />
            );
          }
          if (event.kind === "notice" || event.kind === "attachment") return <NoticeRow key={event.id} event={event} />;
          const clock = eventTime(event, new Date(), i18n.language);
          const human = isHumanMessage(event);
          return (
            <article key={event.id} data-testid={`thread-msg-${event.id}`} className="border-b border-wc-default px-3 py-3">
              <div className="mb-1 flex h-6 min-w-0 items-center gap-2 text-xs text-wc-text-muted">
                <b className="truncate text-[12.5px] font-semibold text-wc-text-secondary">{t(speakerKey(event), speakerParams(event))}</b>
                <time dateTime={event.occurredAt || event.createdAt} title={`#${String(event.sequence)}`} className="shrink-0 font-mono text-[11px] text-wc-text-faint">
                  {clock.label ? `${clock.label} · ` : ""}{clock.timestamp}
                </time>
              </div>
              <div style={{ fontSize: `${String(fontSize)}px` }} className={cn("text-wc-text-primary", human && "rounded-[10px] bg-sky-500/10 px-3 py-2")}>
                <MarkdownRenderer content={event.text} timeAnchor={event.createdAt} imageSource={sessionPreviewSource(sessionId)} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} onMermaidOpen={onMermaidOpen} />
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

function timeOf(event: ConversationEvent | undefined): string | undefined {
  if (!event || isWorkRow(event)) return undefined;
  return event.occurredAt || event.createdAt;
}
