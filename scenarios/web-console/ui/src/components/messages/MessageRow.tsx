import {
  memo,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { useTranslation } from "react-i18next";
import { Loader2, MoreHorizontal } from "lucide-react";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { copyText } from "../../lib/clipboard";
import { MarkdownRenderer } from "../markdown";
import {
  actionIcon,
  actionLabelKey,
  actionPlacement,
  type MessageActionContext,
} from "./messageActions";
import { MessageActionList, useMessageActions, type ActionsOrigin } from "./MessageActionList";
import { messageOutline } from "./outline";
import { speakerKey, timeLabel } from "./speaker";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-row-and-action-reveal

/**
 * Rows are capped at this height before their first paint and end in the
 * reader footer when their content is taller. A row never grows inline, so its
 * height is known up front and the list never jumps when it is measured.
 */
export const COLLAPSE_THRESHOLD_PX = 400;
/** At most this many controls render inline, the overflow trigger included. */
const MAX_INLINE_CONTROLS = 3;

export interface PressHandlers {
  onPointerDown: (event: ReactPointerEvent<HTMLElement>) => void;
  onPointerCancel: () => void;
  onContextMenu: (event: ReactMouseEvent<HTMLElement>) => void;
}

interface MessageRowProps {
  actionContext: MessageActionContext;
  fontSize: number;
  isFocused: boolean;
  isSearchFocused: boolean;
  isDimmed: boolean;
  /** Touch devices reveal actions by tap and open the full list by long-press; nothing reveals on hover. */
  coarsePointer: boolean;
  /** On touch, whether this is the one row the pane has revealed by tap. */
  tapRevealed: boolean;
  /** The pane owns which row's action list is open, so only one is. */
  actionsOpen: boolean;
  actionsOrigin: ActionsOrigin | null;
  onOpenActions: (eventId: string, origin: ActionsOrigin | null) => void;
  onCloseActions: () => void;
  getPressHandlers: (eventId: string) => PressHandlers;
  onLinkClick: (href: string, event: React.MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick: (path: string) => void;
  onMermaidOpen: (code: string) => void;
}

function MessageRowImpl({
  actionContext,
  fontSize,
  isFocused,
  isSearchFocused,
  isDimmed,
  coarsePointer,
  tapRevealed,
  actionsOpen,
  actionsOrigin,
  onOpenActions,
  onCloseActions,
  getPressHandlers,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MessageRowProps) {
  const {
    event,
    isTtsSpeaking,
    activeSpeakingEventId,
    isAudioLoading,
    getSummarizeError,
    onClearSummarizeError,
    isPlaintext,
    onOpenReader,
  } = actionContext;
  const { t, i18n } = useTranslation();
  const [revealed, setRevealed] = useState(false);
  const [isTall, setIsTall] = useState(false);
  const rowRef = useRef<HTMLElement | null>(null);
  const moreButtonRef = useRef<HTMLButtonElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const node = contentRef.current;
    if (!node) return;
    const measure = () => { setIsTall(node.scrollHeight > COLLAPSE_THRESHOLD_PX); };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => { measure(); });
    observer.observe(node);
    return () => { observer.disconnect(); };
  }, [event.text, isPlaintext]);

  const isUser = event.role === "user";
  const isSpeaking = !isUser && isTtsSpeaking && activeSpeakingEventId === event.id;
  const outline = useMemo(() => (isTall ? messageOutline(event.text) : null), [event.text, isTall]);
  const summarizeError = getSummarizeError(event.id);

  const { ctx: resolvedContext, actions, composites } = useMessageActions(actionContext, rowRef, isTall);
  const primary = actions.filter((action) => actionPlacement(action, resolvedContext) === "primary");
  const inline = primary.slice(0, MAX_INLINE_CONTROLS - 1);
  // Hover and focus reveal on a fine pointer; on touch, a tap does (the pane
  // keeps it to one row).
  const showCluster = (coarsePointer ? tapRevealed : revealed) && !actionsOpen;
  const press = coarsePointer ? getPressHandlers(event.id) : null;

  const openFromMoreButton = () => {
    const rect = moreButtonRef.current?.getBoundingClientRect();
    onOpenActions(event.id, rect ? { x: rect.right, y: rect.bottom } : null);
  };

  return (
    <article
      ref={rowRef}
      data-testid={`msg-card-${event.id}`}
      data-focused={isFocused || undefined}
      data-speaking={isSpeaking || undefined}
      // Focusable by code only: the reader returns focus here on close.
      tabIndex={-1}
      className={cn(
        "relative border-b border-wc-default px-3 py-3 transition-colors",
        isSpeaking && "border-l-2 border-l-wc-accent",
        isFocused && "bg-wc-accent/5",
        isSearchFocused && "rounded-r-lg ring-1 ring-wc-accent/50",
        isDimmed && "opacity-40",
      )}
      onMouseEnter={coarsePointer ? undefined : () => { setRevealed(true); }}
      onMouseLeave={coarsePointer ? undefined : () => { setRevealed(false); }}
      onFocus={coarsePointer ? undefined : () => { setRevealed(true); }}
      onBlur={coarsePointer ? undefined : (focusEvent) => {
        if (!focusEvent.currentTarget.contains(focusEvent.relatedTarget as Node | null)) setRevealed(false);
      }}
      onContextMenu={press?.onContextMenu ?? ((menuEvent) => {
        menuEvent.preventDefault();
        onOpenActions(event.id, { x: menuEvent.clientX, y: menuEvent.clientY });
      })}
      onPointerDown={press?.onPointerDown}
      onPointerCancel={press?.onPointerCancel}
    >
      {/* The header line is a fixed height, so revealing the actions in it
          never changes the row's height and never moves the list. The
          buttons' boxes and tap targets overhang it into the row padding. */}
      <div className="mb-1 flex h-6 min-w-0 items-center gap-2 text-xs text-wc-text-muted">
        <b data-testid={`msg-speaker-${event.id}`} className="truncate text-[12.5px] font-semibold text-wc-text-secondary">
          {t(speakerKey(event) as never)}
        </b>
        <time
          data-testid={`msg-time-${event.id}`}
          dateTime={event.createdAt}
          title={`#${String(event.sequence)}`}
          className="shrink-0 font-mono text-[11px] text-wc-text-faint"
        >
          {timeLabel(event.createdAt, new Date(), i18n.language)}
        </time>
        {showCluster && (
          <div data-testid="msg-actions-inline" className="flex shrink-0 items-center">
            {inline.map((action) => {
              const Icon = actionIcon(action, resolvedContext);
              const label = t(actionLabelKey(action, resolvedContext) as never);
              const loading = action.id === "read-from-here" && isAudioLoading;
              if (action.id === "copy") {
                // The library's copy button owns the write and its result; a
                // copy from the action list shows here through `copied`.
                return (
                  <CopyIconButton
                    key={action.id}
                    data-message-action-inline
                    data-testid={action.testId(resolvedContext)}
                    size="xs"
                    value={event.text}
                    writeText={copyText}
                    copied={actionContext.copied}
                    aria-label={label}
                    copiedLabel={t(strings.messageActions.copied)}
                    failedLabel={t(strings.messageActions.copyFailed)}
                  />
                );
              }
              return (
                <IconButton
                  key={action.id}
                  data-message-action-inline
                  data-testid={action.testId(resolvedContext)}
                  size="xs"
                  surface="ghost"
                  onClick={() => { action.run(resolvedContext); }}
                  disabled={action.disabled?.(resolvedContext)}
                  aria-pressed={action.pressed?.(resolvedContext)}
                  aria-label={label}
                >
                  {loading
                    ? <Loader2 data-testid={`msg-audio-loading-${event.id}`} className="animate-spin" />
                    : <Icon />}
                </IconButton>
              );
            })}
            {actions.length > inline.length && (
              <IconButton
                ref={moreButtonRef}
                data-message-action-inline
                data-testid={`msg-actions-more-${event.id}`}
                size="xs"
                surface="ghost"
                aria-label={t(strings.messageActions.more)}
                aria-haspopup="menu"
                onClick={openFromMoreButton}
              >
                <MoreHorizontal />
              </IconButton>
            )}
          </div>
        )}
      </div>

      <div
        className={cn(
          "relative max-h-[400px] overflow-hidden",
          isUser && "rounded-[10px] bg-sky-500/10 px-3 py-2",
        )}
      >
        <div
          ref={contentRef}
          data-testid={`msg-markdown-${event.id}`}
          style={{ fontSize: `${String(fontSize)}px` }}
          className="text-wc-text-primary"
        >
          {isPlaintext ? (
            <pre
              data-testid={`msg-plaintext-${event.id}`}
              className="whitespace-pre-wrap break-words [overflow-wrap:anywhere] font-mono"
            >
              {event.text}
            </pre>
          ) : (
            <MarkdownRenderer content={event.text} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick} onMermaidOpen={onMermaidOpen} />
          )}
        </div>

        {isTall && outline && (
          <div className="absolute inset-x-0 bottom-0 flex items-end bg-gradient-to-t from-wc-surface-base via-wc-surface-base/90 to-transparent pt-16">
            <button
              type="button"
              data-testid={`msg-open-reader-${event.id}`}
              disabled={!onOpenReader}
              onClick={() => { onOpenReader?.(event.id); }}
              className="min-h-11 w-full text-start text-xs text-wc-accent transition hover:text-wc-accent/80 disabled:text-wc-text-faint"
            >
              {t(strings.messagesPane.readerFooter, {
                words: outline.words.toLocaleString(i18n.language),
                headings: outline.headings,
                codeBlocks: outline.codeBlocks,
              })}
            </button>
          </div>
        )}
      </div>

      {summarizeError && (
        <div
          role="alert"
          data-testid={`msg-summarize-error-${event.id}`}
          className="mt-2 flex items-center gap-2 rounded-lg bg-red-500/10 px-3 py-1 text-[11px] text-red-400"
        >
          <span className="min-w-0 flex-1">{summarizeError}</span>
          <button
            type="button"
            data-testid={`msg-clear-summarize-error-${event.id}`}
            onClick={() => { onClearSummarizeError(event.id); }}
            className="min-h-11 shrink-0 px-2 font-medium text-wc-text-muted hover:text-wc-text-primary"
          >
            {t(strings.messagesPane.dismissError)}
          </button>
        </div>
      )}

      {composites}

      {actionsOpen && (
        <MessageActionList
          actions={actions}
          ctx={resolvedContext}
          coarsePointer={coarsePointer}
          origin={actionsOrigin}
          anchorRef={rowRef}
          onClose={onCloseActions}
        />
      )}
    </article>
  );
}

/** Re-renders only when something it shows changed; handlers are stable by contract. */
export const MessageRow = memo(MessageRowImpl, (prev, next) => (
  Object.keys(prev.actionContext).length === Object.keys(next.actionContext).length &&
  Object.entries(prev.actionContext).every(([key, value]) => (
    value === (next.actionContext as unknown as Record<string, unknown>)[key]
  )) &&
  prev.fontSize === next.fontSize &&
  prev.isFocused === next.isFocused &&
  prev.isSearchFocused === next.isSearchFocused &&
  prev.isDimmed === next.isDimmed &&
  prev.coarsePointer === next.coarsePointer &&
  prev.tapRevealed === next.tapRevealed &&
  prev.actionsOpen === next.actionsOpen &&
  prev.actionsOrigin === next.actionsOrigin &&
  prev.onOpenActions === next.onOpenActions &&
  prev.onCloseActions === next.onCloseActions &&
  prev.getPressHandlers === next.getPressHandlers &&
  prev.onLinkClick === next.onLinkClick &&
  prev.onFileReferenceClick === next.onFileReferenceClick &&
  prev.onMermaidOpen === next.onMermaidOpen
));
