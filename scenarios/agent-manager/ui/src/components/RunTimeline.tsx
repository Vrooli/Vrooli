import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  Activity,
  AlertCircle,
  Brain,
  Check,
  ChevronDown,
  ChevronRight,
  Copy,
  FileText,
  Filter,
  Loader2,
  Paperclip,
  Send,
  Sparkles,
  Terminal,
  Trash2,
  Wrench,
} from "lucide-react";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Textarea } from "./ui/textarea";
import { CodeBlock } from "./markdown/components/CodeBlock";
import { MarkdownRenderer } from "./markdown";
import { AttachmentPreview } from "./AttachmentPreview";
import { useAttachments } from "../hooks/useAttachments";
import { useViewportSize } from "../hooks/useViewportSize";
import { getPopoverPosition, type PopoverPlacement } from "../lib/popoverPosition";
import {
  buildTimelineEntries,
  countTimelineEntriesByCategory,
  createDefaultTimelineFilterState,
  createShowAllTimelineFilterState,
  filterTimelineEntries,
  getTimelineCategoryLabel,
  getTimelineEventLabel,
  getTimelineEventSummary,
  getTimelineModeLabel,
  TIMELINE_CATEGORY_ORDER,
  type TimelineCategory,
  type TimelineDisplayMode,
  type TimelineEventEntry,
  type TimelineFilterState,
  type TimelineMessageEntry,
} from "../lib/runTimeline";
import { cn } from "../lib/utils";
import { formatRelativeTimeShort, formatStandardDateTime } from "../lib/dateTime";
import type { Run, RunEvent } from "../types";
import { RunStatus } from "../types";

const FILTER_STORAGE_KEY = "agm.runTimelineFilters.v1";

interface RunTimelineProps {
  run: Run;
  events: RunEvent[];
  eventsLoading: boolean;
  onContinue: (message: string, attachmentIds?: string[]) => Promise<void>;
  onDeleteMessage: (eventId: string) => Promise<void>;
}

interface FilterPanelPosition {
  top: number;
  left: number;
  width: number;
  maxHeight: number;
  placement: PopoverPlacement;
}

function loadPersistedFilters(): TimelineFilterState {
  if (typeof window === "undefined") return createDefaultTimelineFilterState();

  try {
    const raw = window.localStorage.getItem(FILTER_STORAGE_KEY);
    if (!raw) return createDefaultTimelineFilterState();

    const parsed = JSON.parse(raw) as Partial<TimelineFilterState>;
    const defaults = createDefaultTimelineFilterState();
    const mode = parsed.mode;
    return {
      mode: mode === "conversation" || mode === "combined" || mode === "events" ? mode : defaults.mode,
      categories: {
        ...defaults.categories,
        ...(parsed.categories ?? {}),
      },
    };
  } catch {
    return createDefaultTimelineFilterState();
  }
}

export function RunTimeline({
  run,
  events,
  eventsLoading,
  onContinue,
  onDeleteMessage,
}: RunTimelineProps) {
  const [inputMessage, setInputMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [copyStatus, setCopyStatus] = useState<Record<string, "idle" | "copied">>({});
  const [continueError, setContinueError] = useState<string | null>(null);
  const [revealedMessages, setRevealedMessages] = useState<Record<string, boolean>>({});
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filters, setFilters] = useState<TimelineFilterState>(() => loadPersistedFilters());
  const [filterPanelPosition, setFilterPanelPosition] = useState<FilterPanelPosition | null>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const filterPanelRef = useRef<HTMLDivElement>(null);
  const filterButtonRef = useRef<HTMLButtonElement>(null);
  const timelineEndRef = useRef<HTMLDivElement>(null);
  const isNearBottomRef = useRef(true);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { attachments, addAttachment, removeAttachment, clearAttachments, isUploading, getUploadedIds } = useAttachments();
  const { isDesktop } = useViewportSize();

  const timelineEntries = useMemo(() => buildTimelineEntries(events), [events]);
  const visibleEntries = useMemo(() => filterTimelineEntries(timelineEntries, filters), [timelineEntries, filters]);
  const categoryCounts = useMemo(() => countTimelineEntriesByCategory(timelineEntries), [timelineEntries]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(filters));
  }, [filters]);

  useEffect(() => {
    if (!filtersOpen) return;

    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node;
      const clickedPanel = filterPanelRef.current?.contains(target) ?? false;
      const clickedButton = filterButtonRef.current?.contains(target) ?? false;
      if (!clickedPanel && !clickedButton) {
        setFiltersOpen(false);
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setFiltersOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [filtersOpen]);

  useEffect(() => {
    if (!filtersOpen) {
      setFilterPanelPosition(null);
      return;
    }

    const updatePosition = () => {
      const anchor = filterButtonRef.current;
      const panel = filterPanelRef.current;
      if (!anchor || !panel || typeof window === "undefined") return;

      const anchorRect = anchor.getBoundingClientRect();
      const panelRect = panel.getBoundingClientRect();
      const { top, left, width, maxHeight, placement } = getPopoverPosition({
        anchorRect,
        viewportWidth: window.innerWidth,
        viewportHeight: window.innerHeight,
        preferredWidth: 320,
        panelHeight: panelRect.height || 420,
      });

      setFilterPanelPosition({ top, left, width, maxHeight, placement });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [filtersOpen]);

  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const threshold = 80;
    isNearBottomRef.current =
      container.scrollHeight - container.scrollTop - container.clientHeight <= threshold;
  }, []);

  useEffect(() => {
    if (!isNearBottomRef.current) return;
    timelineEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [visibleEntries.length]);

  const isGenerating = useMemo(() => {
    return run.status === RunStatus.RUNNING || run.status === RunStatus.STARTING || run.status === RunStatus.PENDING;
  }, [run.status]);

  const canContinue = useMemo(() => {
    return run.actions?.canContinue ?? false;
  }, [run.actions?.canContinue]);

  useEffect(() => {
    if (!isGenerating || !isNearBottomRef.current) return;
    timelineEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [isGenerating]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    const styles = window.getComputedStyle(textarea);
    const lineHeight = Number.parseFloat(styles.lineHeight || "20");
    const padding =
      Number.parseFloat(styles.paddingTop || "0") + Number.parseFloat(styles.paddingBottom || "0");
    const maxHeight = lineHeight * 10 + padding;
    const nextHeight = Math.min(textarea.scrollHeight, maxHeight);

    textarea.style.height = `${nextHeight}px`;
    textarea.style.overflowY = textarea.scrollHeight > maxHeight ? "auto" : "hidden";
  }, [inputMessage]);

  const handleSend = async () => {
    const hasText = inputMessage.trim().length > 0;
    const hasAttachments = attachments.length > 0;
    if ((!hasText && !hasAttachments) || !canContinue || sending || isUploading) return;

    setSending(true);
    setContinueError(null);
    try {
      const ids = getUploadedIds();
      await onContinue(inputMessage.trim(), ids.length > 0 ? ids : undefined);
      setInputMessage("");
      clearAttachments();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to continue run";
      setContinueError(message);
    } finally {
      setSending(false);
    }
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      addAttachment(file);
      event.target.value = "";
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey && isDesktop) {
      event.preventDefault();
      void handleSend();
    }
  };

  const handleCopy = async (messageId: string, content: string) => {
    try {
      await navigator.clipboard.writeText(content);
      setCopyStatus((prev) => ({ ...prev, [messageId]: "copied" }));
      setTimeout(() => {
        setCopyStatus((prev) => ({ ...prev, [messageId]: "idle" }));
      }, 2000);
    } catch {
      // Ignore clipboard failures and preserve current UI state.
    }
  };

  const handleDelete = async (messageId: string) => {
    if (!window.confirm("Delete this message? It will be hidden but remain in the run history.")) {
      return;
    }

    try {
      await onDeleteMessage(messageId);
      setRevealedMessages((prev) => {
        if (!prev[messageId]) return prev;
        const next = { ...prev };
        delete next[messageId];
        return next;
      });
    } catch {
      // Keep the existing timeline visible even if deletion fails.
    }
  };

  const updateMode = (mode: TimelineDisplayMode) => {
    setFilters((prev) => ({ ...prev, mode }));
  };

  const updateCategory = (category: TimelineCategory, checked: boolean) => {
    setFilters((prev) => ({
      ...prev,
      categories: { ...prev.categories, [category]: checked },
    }));
  };

  const resetFilters = () => {
    setFilters(createDefaultTimelineFilterState());
  };

  const enableAllFilters = () => {
    setFilters(createShowAllTimelineFilterState());
  };

  const activeCategoryLabels = TIMELINE_CATEGORY_ORDER
    .filter((category) => filters.categories[category])
    .map((category) => getTimelineCategoryLabel(category));

  return (
    <>
      <div className="relative flex h-full flex-col">
        <div className="pointer-events-none absolute right-3 top-2.5 z-10 sm:right-4">
          <Button
            ref={filterButtonRef}
            type="button"
            variant="outline"
            size="icon"
            className="pointer-events-auto relative h-9 w-9 rounded-full shadow-sm"
            aria-expanded={filtersOpen}
            aria-haspopup="dialog"
            aria-label={`Open timeline filters. ${getTimelineModeLabel(filters.mode)} mode. ${visibleEntries.length} of ${timelineEntries.length} entries visible.`}
            onClick={() => setFiltersOpen((prev) => !prev)}
          >
            <Filter className="h-4 w-4" />
            <span className="absolute -right-1 -top-1 min-w-5 rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-semibold leading-none text-primary-foreground">
              {timelineEntries.length}
            </span>
          </Button>
        </div>
        <div ref={scrollContainerRef} onScroll={handleScroll} className="flex-1 min-h-0 overflow-y-auto p-3 sm:p-4">
          {eventsLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">Loading timeline...</span>
            </div>
          ) : visibleEntries.length === 0 ? (
            <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
              <Filter className="h-10 w-10 text-muted-foreground/40" />
              <div>
                <p className="font-medium text-muted-foreground">No timeline entries match the current filters</p>
                <p className="mt-1 text-sm text-muted-foreground/80">
                  Reset to the hybrid view to see messages, tool use, reasoning, and errors together.
                </p>
              </div>
              <Button type="button" variant="outline" onClick={resetFilters}>
                Reset Filters
              </Button>
            </div>
          ) : (
            <div>
              {visibleEntries.map((entry, index) => {
                const previousEntry = index > 0 ? visibleEntries[index - 1] : null;
                const spacingClass = getTimelineEntrySpacing(previousEntry, entry);

                return (
                  <div key={entry.id} className={spacingClass}>
                    {entry.kind === "message" ? (
                      <MessageBubble
                        entry={entry}
                        copyState={copyStatus[entry.id] ?? "idle"}
                        isRevealed={revealedMessages[entry.id] ?? false}
                        onCopy={handleCopy}
                        onDelete={handleDelete}
                        onToggleReveal={(messageId) => {
                          setRevealedMessages((prev) => ({ ...prev, [messageId]: !prev[messageId] }));
                        }}
                      />
                    ) : (
                      <TimelineEventRow entry={entry} />
                    )}
                  </div>
                );
              })}

              {isGenerating ? (
                <div className="rounded-lg border border-dashed border-border bg-muted/30 px-4 py-3">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span>Run is still generating new timeline entries...</span>
                  </div>
                </div>
              ) : null}

              <div ref={timelineEndRef} />
            </div>
          )}
        </div>

        {canContinue ? (
          <div className="border-t border-border px-3 py-4 sm:px-4">
            {attachments.length > 0 ? (
              <AttachmentPreview
                attachments={attachments}
                onRemove={removeAttachment}
                isUploading={isUploading}
              />
            ) : null}

            <div className="flex gap-2">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="self-end"
                onClick={() => fileInputRef.current?.click()}
                title="Attach image"
              >
                <Paperclip className="h-4 w-4" />
              </Button>

              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                onChange={handleFileSelect}
                className="hidden"
              />

              <Textarea
                ref={textareaRef}
                value={inputMessage}
                onChange={(event) => {
                  setInputMessage(event.target.value);
                  if (continueError) setContinueError(null);
                }}
                onKeyDown={handleKeyDown}
                placeholder="Type your follow-up message..."
                className="min-h-[60px] resize-none"
                disabled={sending}
              />

              <Button
                type="button"
                onClick={() => void handleSend()}
                disabled={(!inputMessage.trim() && attachments.length === 0) || sending || isUploading}
                className="self-end"
              >
                {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              </Button>
            </div>

            {isDesktop ? (
              <p className="mt-2 text-xs text-muted-foreground">
                Press Enter to send, Shift+Enter for a new line
              </p>
            ) : null}

            {continueError ? (
              <div className="mt-2 flex items-center gap-2 text-xs text-destructive">
                <AlertCircle className="h-3 w-3" />
                <span>{continueError}</span>
              </div>
            ) : null}
          </div>
        ) : (
          <div className="border-t border-border px-3 py-4 text-sm text-muted-foreground sm:px-4">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              <span>{run.actions?.canContinueReason || "Continuation not available"}</span>
            </div>
          </div>
        )}
      </div>
      {filtersOpen && typeof document !== "undefined"
        ? createPortal(
            <div
              ref={filterPanelRef}
              className={cn(
                "fixed z-50 flex flex-col overflow-hidden rounded-lg border border-border bg-background shadow-xl",
                filterPanelPosition?.placement === "top" ? "origin-bottom-right" : "origin-top-right",
                filterPanelPosition ? "opacity-100" : "opacity-0"
              )}
              style={{
                top: filterPanelPosition?.top ?? 8,
                left: filterPanelPosition?.left ?? 8,
                width: filterPanelPosition?.width ?? 320,
                maxHeight: filterPanelPosition?.maxHeight ?? 420,
                visibility: filterPanelPosition ? "visible" : "hidden",
              }}
            >
              <div className="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
                <div>
                  <h4 className="text-sm font-semibold">Timeline Filters</h4>
                  <p className="text-xs text-muted-foreground">
                    Choose how much execution detail stays in view.
                  </p>
                </div>
                <Button type="button" variant="ghost" size="sm" onClick={() => setFiltersOpen(false)}>
                  Done
                </Button>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto p-4">
                <div className="space-y-2">
                  <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Display
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    {(["conversation", "combined", "events"] as const).map((mode) => (
                      <button
                        key={mode}
                        type="button"
                        onClick={() => updateMode(mode)}
                        className={cn(
                          "rounded-md border px-3 py-2 text-left text-xs transition-colors",
                          filters.mode === mode
                            ? "border-primary bg-primary/10 text-foreground"
                            : "border-border bg-card text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                        )}
                      >
                        <div className="font-medium">{getTimelineModeLabel(mode)}</div>
                        <div className="mt-1 text-[11px] leading-snug opacity-80">
                          {mode === "conversation"
                            ? "Just message bubbles"
                            : mode === "combined"
                              ? "Messages plus selected events"
                              : "Structured event feed only"}
                        </div>
                      </button>
                    ))}
                  </div>
                </div>

                <div className="mt-4 flex items-center justify-between gap-2">
                  <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Event Types
                  </div>
                  <div className="flex items-center gap-1">
                    <Button type="button" variant="ghost" size="sm" onClick={resetFilters}>
                      Hybrid
                    </Button>
                    <Button type="button" variant="ghost" size="sm" onClick={enableAllFilters}>
                      All
                    </Button>
                  </div>
                </div>

                <div className="mt-2 space-y-2 pr-1">
                  {TIMELINE_CATEGORY_ORDER.map((category) => (
                    <label
                      key={category}
                      className="flex cursor-pointer items-start gap-3 rounded-md border border-transparent px-2 py-2 hover:bg-muted/50"
                    >
                      <Checkbox
                        checked={filters.categories[category]}
                        onCheckedChange={(checked) => updateCategory(category, checked === true)}
                        className="mt-0.5"
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-sm font-medium">{getTimelineCategoryLabel(category)}</span>
                          <span className="text-xs text-muted-foreground">{categoryCounts[category]}</span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {categoryDescription(category)}
                        </p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </div>,
            document.body
          )
        : null}
    </>
  );
}

interface MessageBubbleProps {
  entry: TimelineMessageEntry;
  copyState: "idle" | "copied";
  isRevealed: boolean;
  onCopy: (messageId: string, content: string) => void;
  onDelete: (messageId: string) => void;
  onToggleReveal: (messageId: string) => void;
}

function MessageBubble({
  entry,
  copyState,
  isRevealed,
  onCopy,
  onDelete,
  onToggleReveal,
}: MessageBubbleProps) {
  const showContent = !entry.deleted || isRevealed;

  return (
    <div
      className={cn(
        "flex min-w-0",
        entry.role === "user"
          ? "flex-row-reverse"
          : entry.role === "system"
            ? "justify-center"
            : "flex-row"
      )}
    >
      <div
        className={cn(
          "flex min-w-0 flex-col gap-1",
          entry.role === "user"
            ? "items-end"
            : entry.role === "system"
              ? "items-center"
              : "items-start"
        )}
      >
        <div
          className={cn(
            "max-w-[95%] overflow-hidden rounded-lg px-4 py-3",
            entry.role === "user"
              ? "bg-primary text-primary-foreground"
              : entry.role === "system"
                ? "border border-primary/25 bg-primary/5"
                : "bg-muted"
          )}
        >
          <div className="mb-2 flex items-center gap-2 text-[11px] uppercase tracking-wide text-muted-foreground">
            <span className={cn(entry.role === "user" ? "text-primary-foreground/70" : "")}>
              {entry.role === "user" ? "You" : entry.role === "assistant" ? "Assistant" : "System"}
            </span>
            <span className={cn(entry.role === "user" ? "text-primary-foreground/70" : "")}>
              {formatRelativeTimeShort(entry.event.timestamp)}
            </span>
          </div>

          {showContent ? (
            <div className="text-sm break-words overflow-x-auto">
              {entry.attachments.length > 0 ? (
                <div className="mb-2 flex flex-wrap gap-2">
                  {entry.attachments.map((attachment) => (
                    <a
                      key={attachment.id}
                      href={attachment.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block"
                    >
                      <img
                        src={attachment.url}
                        alt={attachment.fileName}
                        className="max-h-[200px] max-w-[200px] rounded border border-border object-cover"
                      />
                    </a>
                  ))}
                </div>
              ) : null}

              {entry.content ? <MarkdownRenderer content={entry.content} /> : null}
            </div>
          ) : (
            <div className="text-sm italic text-muted-foreground">Message deleted</div>
          )}

          <div
            className={cn(
              "mt-2 text-[10px]",
              entry.role === "user" ? "text-primary-foreground/70" : "text-muted-foreground"
            )}
          >
            {formatStandardDateTime(entry.event.timestamp)}
          </div>
        </div>

        <div
          className={cn(
            "flex items-center gap-1",
            entry.role === "user"
              ? "justify-end"
              : entry.role === "system"
                ? "justify-center"
                : "justify-start"
          )}
        >
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onCopy(entry.id, entry.content)}
            disabled={!showContent}
            title="Copy message"
          >
            {copyState === "copied" ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          </Button>

          {entry.deleted ? (
            <Button type="button" variant="link" size="sm" className="px-2" onClick={() => onToggleReveal(entry.id)}>
              {isRevealed ? "Hide message" : "Show message"}
            </Button>
          ) : (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive"
              onClick={() => void onDelete(entry.id)}
              title="Delete message"
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

function TimelineEventRow({ entry }: { entry: TimelineEventEntry }) {
  const [expanded, setExpanded] = useState(false);
  const icon = getEventIcon(entry.category);
  const payloadValue = entry.event.data.value as Record<string, unknown>;

  return (
    <div
      className={cn(
        "overflow-hidden border-b border-border/60 border-l-2 bg-transparent text-xs transition-colors",
        getEventChrome(entry.category)
      )}
    >
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-1.5 text-left"
        onClick={() => setExpanded((prev) => !prev)}
      >
        <span className={cn("shrink-0", getEventIconColor(entry.category))}>{icon}</span>
        <span className="shrink-0 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
          {getTimelineEventLabel(entry)}
        </span>
        <div className="min-w-0 flex-1 truncate text-sm text-foreground">
          {getTimelineEventSummary(entry)}
        </div>
        <span className="shrink-0 text-[10px] text-muted-foreground">
          {formatRelativeTimeShort(entry.event.timestamp)}
        </span>
        {expanded ? <ChevronDown className="h-3 w-3 text-muted-foreground" /> : <ChevronRight className="h-3 w-3 text-muted-foreground" />}
      </button>

      {expanded ? (
        <div className="border-t border-border/60 bg-muted/10 px-3 pb-2 pt-1.5">
          <div className="mb-2 flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
            <span className="rounded-full border border-border bg-background px-2 py-0.5 font-medium uppercase tracking-wide">
              {getTimelineCategoryLabel(entry.category)}
            </span>
            <span>{formatStandardDateTime(entry.event.timestamp)}</span>
            <span>Seq {String(typeof entry.event.sequence === "bigint" ? Number(entry.event.sequence) : entry.event.sequence ?? 0)}</span>
          </div>
          <CodeBlock code={JSON.stringify(payloadValue, null, 2)} language="json" />
        </div>
      ) : null}
    </div>
  );
}

function getEventIcon(category: TimelineEventEntry["category"]) {
  switch (category) {
    case "reasoning":
      return <Brain className="h-3.5 w-3.5" />;
    case "tools":
      return <Wrench className="h-3.5 w-3.5" />;
    case "errors":
      return <AlertCircle className="h-3.5 w-3.5" />;
    case "status":
      return <Activity className="h-3.5 w-3.5" />;
    case "artifacts":
      return <FileText className="h-3.5 w-3.5" />;
    case "metrics":
      return <Sparkles className="h-3.5 w-3.5" />;
    case "compaction":
      return <Sparkles className="h-3.5 w-3.5" />;
    case "redactions":
      return <Trash2 className="h-3.5 w-3.5" />;
    default:
      return <Terminal className="h-3.5 w-3.5" />;
  }
}

function getEventChrome(category: TimelineEventEntry["category"]): string {
  switch (category) {
    case "reasoning":
      return "border-l-sky-400";
    case "tools":
      return "border-l-amber-400";
    case "errors":
      return "border-l-destructive";
    case "status":
      return "border-l-primary";
    case "compaction":
      return "border-l-emerald-500";
    case "redactions":
      return "border-l-rose-500";
    default:
      return "border-l-muted-foreground";
  }
}

function getEventIconColor(category: TimelineEventEntry["category"]): string {
  switch (category) {
    case "reasoning":
      return "text-sky-600 dark:text-sky-300";
    case "tools":
      return "text-amber-600 dark:text-amber-300";
    case "errors":
      return "text-destructive";
    case "status":
      return "text-primary";
    case "compaction":
      return "text-emerald-600 dark:text-emerald-300";
    case "redactions":
      return "text-rose-600 dark:text-rose-300";
    default:
      return "text-muted-foreground";
  }
}

function categoryDescription(category: TimelineCategory): string {
  switch (category) {
    case "messages":
      return "User, assistant, and system messages rendered as chat bubbles.";
    case "reasoning":
      return "Thinking and reasoning traces emitted by supported runners.";
    case "tools":
      return "Tool calls, file operations, and tool execution results.";
    case "errors":
      return "Execution failures and rate limiting events.";
    case "status":
      return "Run state changes and progress updates.";
    case "logs":
      return "Operational logs that are not classified as reasoning.";
    case "artifacts":
      return "Diffs, logs, screenshots, and other created artifacts.";
    case "metrics":
      return "Token, cost, and other telemetry updates.";
    case "compaction":
      return "Context compaction and summarization markers.";
    case "redactions":
      return "Message deletion and redaction markers.";
  }
}

function getTimelineEntrySpacing(
  previousEntry: TimelineMessageEntry | TimelineEventEntry | null | undefined,
  entry: TimelineMessageEntry | TimelineEventEntry
): string {
  if (!previousEntry) return "";
  if (previousEntry.kind === "event" && entry.kind === "event") return "";
  return "pt-3";
}
