// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useRef, useState, useEffect } from "react";
import { Maximize2, Minimize2, ChevronUp, ChevronDown } from "lucide-react";
import { TOOLBAR_KEYS, type ToolbarKey } from "../consts/toolbar-keys";
import { cn } from "../lib/classnames";
import { slugify } from "../lib/slugify";
import { useDraftPersistence } from "../hooks/useDraftPersistence";
import { useCommandHistory } from "../hooks/useCommandHistory";

// [REQ:P0-007a] Floating Toolbar Component
// [REQ:P0-007b] Terminal Key/Chord Mapping

/** Max visible lines before the textarea stops growing. */
const MAX_VISIBLE_LINES = 4;
/** Approximate line height in px for the textarea. */
const LINE_HEIGHT_PX = 20;
/** Max textarea height: MAX_VISIBLE_LINES * line-height + padding. */
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;

type SendStatus = "sent" | "queued" | "idle";

interface MobileToolbarProps {
  /** Callback to inject input into the active terminal. Returns true if sent immediately. */
  onInput: (data: string) => boolean;
  /** Whether the toolbar is visible. */
  visible?: boolean;
}

export default function MobileToolbar({
  onInput,
  visible = true,
}: MobileToolbarProps) {
  const { value: inputValue, setValue: setInputValue, clearDraft } = useDraftPersistence();
  const { push: pushHistory, navigateUp, navigateDown, resetNavigation } = useCommandHistory();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [sendStatus, setSendStatus] = useState<SendStatus>("idle");
  // Store draft when browsing history so we can restore it
  const savedDraftRef = useRef<string | null>(null);
  const statusTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clear status timer on unmount
  useEffect(() => {
    return () => {
      if (statusTimerRef.current !== null) clearTimeout(statusTimerRef.current);
    };
  }, []);

  // Auto-resize textarea height
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    // Reset to auto to measure scrollHeight correctly
    el.style.height = "auto";
    const target = expanded
      ? el.scrollHeight
      : Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT);
    el.style.height = `${target}px`;
  }, [inputValue, expanded]);

  const showStatus = useCallback((status: SendStatus) => {
    setSendStatus(status);
    if (statusTimerRef.current !== null) clearTimeout(statusTimerRef.current);
    if (status !== "idle") {
      statusTimerRef.current = setTimeout(() => {
        statusTimerRef.current = null;
        setSendStatus("idle");
      }, 2500);
    }
  }, []);

  const refocusTextarea = useCallback(() => {
    // Use setTimeout to ensure focus happens after the current event cycle,
    // which is necessary on mobile to reliably re-grab the virtual keyboard.
    setTimeout(() => textareaRef.current?.focus(), 0);
  }, []);

  const handleKey = useCallback(
    (key: ToolbarKey) => {
      onInput(key.input);
      refocusTextarea();
    },
    [onInput, refocusTextarea],
  );

  const submitCommand = useCallback(() => {
    const trimmed = inputValue.trim();
    if (!trimmed) return;

    const sent = onInput(inputValue + "\n");
    if (sent) {
      pushHistory(trimmed);
      clearDraft();
      showStatus("sent");
    } else {
      // Data was queued, not sent — keep input so user can copy/retry
      pushHistory(trimmed);
      showStatus("queued");
    }
    resetNavigation();
    savedDraftRef.current = null;
    refocusTextarea();
  }, [inputValue, onInput, pushHistory, clearDraft, showStatus, resetNavigation, refocusTextarea]);

  const handleHistoryNavigation = useCallback(
    (direction: "up" | "down") => {
      // Save current draft before first navigation
      if (savedDraftRef.current === null) {
        savedDraftRef.current = inputValue;
      }
      const entry = direction === "up" ? navigateUp() : navigateDown();
      if (entry !== null) {
        setInputValue(entry);
      } else if (direction === "down") {
        // Returned to draft
        setInputValue(savedDraftRef.current ?? "");
        savedDraftRef.current = null;
      }
    },
    [inputValue, navigateUp, navigateDown, setInputValue],
  );

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      className="flex shrink-0 flex-col border-t border-wc-default bg-wc-surface-raised md:hidden touch-manipulation"
    >
      {/* Command input row */}
      <div className="flex items-end gap-1 px-2 py-1.5">
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <textarea
            ref={textareaRef}
            data-testid="mobile-command-input"
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              resetNavigation();
              savedDraftRef.current = null;
            }}
            autoComplete="off"
            autoCorrect="on"
            spellCheck={false}
            rows={1}
            placeholder="Type command…"
            className={cn(
              "min-w-0 resize-none rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-sm text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent",
              expanded ? "overflow-y-auto" : "overflow-hidden",
            )}
            style={{
              lineHeight: `${LINE_HEIGHT_PX}px`,
              maxHeight: expanded ? "60vh" : `${MAX_TEXTAREA_HEIGHT}px`,
            }}
          />
          {/* Send status indicator */}
          {sendStatus === "queued" && (
            <span data-testid="send-status-queued" className="px-1 text-[10px] text-yellow-400">
              Queued — connection lost. Input preserved.
            </span>
          )}
        </div>
        {/* History navigation */}
        <div className="flex shrink-0 flex-col gap-0.5">
          <button
            data-testid="history-up"
            onPointerDown={(e) => e.preventDefault()}
            onClick={() => handleHistoryNavigation("up")}
            className="rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
            title="Previous command"
          >
            <ChevronUp className="h-3 w-3" />
          </button>
          <button
            data-testid="history-down"
            onPointerDown={(e) => e.preventDefault()}
            onClick={() => handleHistoryNavigation("down")}
            className="rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
            title="Next command"
          >
            <ChevronDown className="h-3 w-3" />
          </button>
        </div>
        {/* Expand/collapse toggle */}
        <button
          data-testid="expand-toggle"
          onPointerDown={(e) => e.preventDefault()}
          onClick={() => setExpanded((prev) => !prev)}
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
          title={expanded ? "Collapse editor" : "Expand editor"}
        >
          {expanded ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
        </button>
        {/* Send button */}
        <button
          data-testid="mobile-command-submit"
          onPointerDown={(e) => e.preventDefault()}
          onClick={submitCommand}
          disabled={!inputValue.trim()}
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-3 py-1.5 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active disabled:opacity-40 touch-manipulation"
        >
          Send
        </button>
      </div>
      {/* Toolbar keys row */}
      <div className="flex items-center gap-1 overflow-x-auto px-2 py-1.5 touch-manipulation">
        {TOOLBAR_KEYS.map((key) => (
          <button
            key={key.label}
            data-testid={`toolbar-key-${slugify(key.label)}`}
            onPointerDown={(e) => e.preventDefault()}
            onClick={() => handleKey(key)}
            className={cn(
              "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation",
              key.width === "wide" ? "min-w-[4rem]" : key.width === "narrow" ? "min-w-[2rem]" : "min-w-[2.5rem]",
            )}
          >
            {key.label}
          </button>
        ))}
      </div>
    </div>
  );
}
