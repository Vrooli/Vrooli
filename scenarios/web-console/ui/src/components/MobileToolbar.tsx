// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useRef, useState, useEffect } from "react";
import { Maximize2, Minimize2, SendHorizontal } from "lucide-react";
import { TOOLBAR_KEYS, type ToolbarKey } from "../consts/toolbar-keys";
import { cn } from "../lib/classnames";
import { slugify } from "../lib/slugify";
import { useDraftPersistence } from "../hooks/useDraftPersistence";

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
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [sendStatus, setSendStatus] = useState<SendStatus>("idle");
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
      clearDraft();
      showStatus("sent");
    } else {
      showStatus("queued");
    }
    refocusTextarea();
  }, [inputValue, onInput, clearDraft, showStatus, refocusTextarea]);

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      className="flex shrink-0 flex-col border-t border-wc-default bg-wc-surface-raised md:hidden touch-manipulation"
    >
      {/* Command input row */}
      <div className="flex items-end gap-0.5 px-1 py-1">
        <div className="flex min-w-0 flex-1 flex-col">
          <textarea
            ref={textareaRef}
            data-testid="mobile-command-input"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            autoComplete="off"
            autoCorrect="on"
            spellCheck={false}
            rows={1}
            placeholder="Type command…"
            className={cn(
              "min-w-0 resize-none rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent",
              expanded ? "overflow-y-auto" : "overflow-hidden",
            )}
            style={{
              lineHeight: `${LINE_HEIGHT_PX}px`,
              maxHeight: expanded ? "60vh" : `${MAX_TEXTAREA_HEIGHT}px`,
            }}
          />
          {sendStatus === "queued" && (
            <span data-testid="send-status-queued" className="px-1 text-[10px] text-yellow-400">
              Queued — connection lost. Input preserved.
            </span>
          )}
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
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active disabled:opacity-40 touch-manipulation"
          title="Send command"
        >
          <SendHorizontal className="h-3.5 w-3.5" />
        </button>
      </div>
      {/* Toolbar keys row */}
      <div className="flex items-center gap-0.5 overflow-x-auto px-1 py-1 touch-manipulation">
        {TOOLBAR_KEYS.map((key) => (
          <button
            key={key.label}
            data-testid={`toolbar-key-${slugify(key.label)}`}
            onPointerDown={(e) => e.preventDefault()}
            onClick={() => handleKey(key)}
            className={cn(
              "shrink-0 rounded border border-wc-default bg-wc-surface-input px-1.5 py-1 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation",
              key.width === "wide" ? "min-w-[3.5rem]" : key.width === "narrow" ? "min-w-[1.75rem]" : "min-w-[2.25rem]",
            )}
          >
            {key.label}
          </button>
        ))}
      </div>
    </div>
  );
}
