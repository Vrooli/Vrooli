// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useRef, useState, useEffect, forwardRef, useImperativeHandle } from "react";
import { Image, Maximize2, Minimize2, SendHorizontal } from "lucide-react";
import { TOOLBAR_KEYS, ESC_KEY, TAB_KEY, ENTER_KEY, ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, type ToolbarKey, applyModifiers } from "../consts/toolbar-keys";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import KeyComboPicker from "./KeyComboPicker";
import VoiceMicButton from "./VoiceMicButton";
import type { StartRecordingOpts } from "../hooks/useVoiceInput";
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

export interface MobileToolbarHandle {
  /** Append text to the command input (used by voice transcription on mobile). */
  appendText: (text: string) => void;
}

interface MobileToolbarProps {
  /** Callback to inject input into the active terminal. Returns true if sent immediately. */
  onInput: (data: string) => boolean;
  /** Move focus to the active terminal (e.g. after submitting a command). */
  onFocusTerminal?: () => void;
  /** Active session ID for per-tab draft persistence. */
  activeSessionId?: string | null;
  /** Whether the toolbar is visible. */
  visible?: boolean;
  // Voice input props (optional - hidden when undefined)
  voiceSupported?: boolean;
  voiceRecording?: boolean;
  voiceTranscribing?: boolean;
  voiceError?: string | null;
  /** 0–1 audio level for live mic visualization */
  voiceLevel?: number;
  voicePartialTranscript?: string;
  voiceBackend?: string;
  onVoiceStart?: (opts?: StartRecordingOpts) => void;
  onVoiceStop?: () => void;
  onUploadImage?: () => void;
}

export default forwardRef<MobileToolbarHandle, MobileToolbarProps>(function MobileToolbar({
  onInput,
  onFocusTerminal,
  activeSessionId,
  visible = true,
  voiceSupported,
  voiceRecording,
  voiceTranscribing,
  voiceError,
  voiceLevel,
  voicePartialTranscript,
  voiceBackend,
  onVoiceStart,
  onVoiceStop,
  onUploadImage,
}, ref) {
  const { value: inputValue, setValue: setInputValue, clearDraft } = useDraftPersistence(activeSessionId);

  useImperativeHandle(ref, () => ({
    appendText: (text: string) => {
      setInputValue(prev => {
        const needsSpace = prev.length > 0 && !prev.endsWith(" ");
        return prev + (needsSpace ? " " : "") + text;
      });
    },
  }), [setInputValue]);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [sendStatus, setSendStatus] = useState<SendStatus>("idle");
  const toolbarLayout = useWorkspaceStore((s) => s.toolbarLayout);
  const modifiers = useWorkspaceStore((s) => s.modifiers);
  const toggleModifier = useWorkspaceStore((s) => s.toggleModifier);
  const clearModifiers = useWorkspaceStore((s) => s.clearModifiers);
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

  const handleKey = useCallback(
    (key: ToolbarKey) => {
      const mods = useWorkspaceStore.getState().modifiers;
      const { data, consumed } = applyModifiers(key.input, mods);
      onInput(data);
      if (consumed) clearModifiers();
      // Re-focus the terminal after sending the key. On mobile, rapid taps can
      // cause the browser to blur the terminal (moving activeElement to body),
      // which dismisses the virtual keyboard. Since the user is actively pressing
      // toolbar keys, they clearly want to interact with the terminal, so
      // restoring focus is always correct here. This does NOT focus the
      // MobileToolbar's textarea — it focuses the xterm.js terminal.
      onFocusTerminal?.();
    },
    [onInput, clearModifiers, onFocusTerminal],
  );

  const submitCommand = useCallback(() => {
    const trimmed = inputValue.trim();
    if (!trimmed) return;

    const mods = useWorkspaceStore.getState().modifiers;
    const hasModifier = mods.ctrl || mods.alt || mods.shift;
    let dataToSend: string;

    if (hasModifier) {
      // With modifiers active, apply them to the input text character by character
      // (useful for combos like Ctrl+C, Ctrl+Alt+2, etc.)
      const { data } = applyModifiers(inputValue, mods);
      dataToSend = data;
      clearModifiers();
    } else {
      // Send exactly what the user typed — no appended newline.
      // The user can explicitly include a newline via the Enter toolbar key
      // if needed. Appending one automatically caused unwanted extra blank
      // lines in the terminal output.
      dataToSend = inputValue;
    }

    const sent = onInput(dataToSend);
    if (sent) {
      clearDraft();
      showStatus("sent");
    } else {
      showStatus("queued");
    }
    // After submitting a command, focus the terminal so the user can
    // immediately see and interact with the output.
    onFocusTerminal?.();
  }, [inputValue, onInput, clearDraft, showStatus, onFocusTerminal, clearModifiers]);

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
              "overflow-y-auto",
              !expanded && "overflow-x-hidden",
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
      {/* Toolbar keys area.
         Focus-preservation strategy (multiple layers to handle browser inconsistencies):
         1. tabIndex={-1} on buttons: makes them non-focusable so they can't steal focus.
         2. onPointerDown preventDefault on buttons: prevents default pointer focus behavior.
         3. onMouseDown preventDefault on container: catches mobile compatibility mouse
            events that slip through the pointerdown handler on rapid double-taps.
         4. select-none: prevents double-tap text selection which can blur the terminal.
         5. handleKey calls onFocusTerminal: restores terminal focus as a safety net
            in case the browser still manages to blur the terminal despite layers 1-4. */}
      {toolbarLayout === "expanded" ? (
        /* ── Expanded layout: two rows with D-pad arrow cluster ──
           ┌────────────────────────────────────────────────────────────┐
           │ [Ctrl] [Alt] [Shift] │     [↑]      │ [📷] │            │
           │ [Esc]  [Tab] [Enter] │ [←] [↓] [→]  │      │    [🎤]   │
           └────────────────────────────────────────────────────────────┘
           The mic button spans both rows for easy access. */
        // eslint-disable-next-line jsx-a11y/no-static-element-interactions -- mousedown prevention is intentional to preserve terminal focus
        <div
          className="grid gap-0.5 px-1 py-1 touch-manipulation select-none"
          style={{ gridTemplateColumns: "auto auto 1fr auto", gridTemplateRows: "auto auto" }}
          onMouseDown={(e) => e.preventDefault()}
        >
          {/* Column 1: Combo picker + Modifiers (row 1) + Special keys (row 2) */}
          <div className="flex flex-col gap-0.5" style={{ gridRow: "1 / -1" }}>
            <div className="flex items-center gap-0.5">
              <KeyComboPicker onInput={onInput} onFocusTerminal={onFocusTerminal} />
              <div className="w-px h-5 bg-wc-default shrink-0" />
              {(["ctrl", "alt", "shift"] as const).map((mod) => (
                <button
                  key={mod}
                  data-testid={`toolbar-mod-${mod}`}
                  tabIndex={-1}
                  onPointerDown={(e) => e.preventDefault()}
                  onClick={() => toggleModifier(mod)}
                  className={cn(
                    "shrink-0 rounded border px-2 py-1.5 text-sm font-medium transition touch-manipulation",
                    modifiers[mod]
                      ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                      : "border-wc-default bg-wc-surface-input text-wc-text-secondary active:bg-wc-accent-active",
                  )}
                >
                  {mod.charAt(0).toUpperCase() + mod.slice(1)}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-0.5">
              {[ESC_KEY, TAB_KEY, ENTER_KEY].map((key) => (
                <button
                  key={key.label}
                  data-testid={`toolbar-key-${slugify(key.label)}`}
                  tabIndex={-1}
                  onPointerDown={(e) => e.preventDefault()}
                  onClick={() => handleKey(key)}
                  className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.75rem]"
                >
                  {key.label}
                </button>
              ))}
            </div>
          </div>

          {/* Column 2: D-pad arrow cluster */}
          <div className="flex flex-col items-center gap-0.5 px-1" style={{ gridRow: "1 / -1" }}>
            {/* Row 1: Up arrow centered */}
            <div className="flex justify-center">
              <button
                data-testid={`toolbar-key-${slugify(ARROW_UP.label)}`}
                tabIndex={-1}
                onPointerDown={(e) => e.preventDefault()}
                onClick={() => handleKey(ARROW_UP)}
                className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]"
              >
                {ARROW_UP.label}
              </button>
            </div>
            {/* Row 2: Left, Down, Right */}
            <div className="flex items-center gap-0.5">
              {[ARROW_LEFT, ARROW_DOWN, ARROW_RIGHT].map((key) => (
                <button
                  key={key.label}
                  data-testid={`toolbar-key-${slugify(key.label)}`}
                  tabIndex={-1}
                  onPointerDown={(e) => e.preventDefault()}
                  onClick={() => handleKey(key)}
                  className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]"
                >
                  {key.label}
                </button>
              ))}
            </div>
          </div>

          {/* Column 3: Image upload button (top row only, right-aligned) */}
          <div className="flex items-start justify-end" style={{ gridColumn: 3, gridRow: 1 }}>
            {onUploadImage && (
              <button
                data-testid="toolbar-upload-image"
                tabIndex={-1}
                onPointerDown={(e) => e.preventDefault()}
                onClick={onUploadImage}
                className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-2 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
                title="Upload image"
              >
                <Image className="h-4 w-4" />
              </button>
            )}
          </div>

          {/* Column 4: Voice mic button spanning both rows for easy access */}
          {voiceSupported && onVoiceStart && onVoiceStop && (
            <div className="flex items-stretch" style={{ gridColumn: 4, gridRow: "1 / -1" }}>
              <VoiceMicButton
                supported={voiceSupported}
                isRecording={voiceRecording ?? false}
                isTranscribing={voiceTranscribing ?? false}
                error={voiceError ?? null}
                audioLevel={voiceLevel}
                partialTranscript={voicePartialTranscript}
                backend={voiceBackend}
                onStart={onVoiceStart}
                onStop={onVoiceStop}
                className="h-full"
                buttonClassName="h-full px-3"
              />
            </div>
          )}
        </div>
      ) : (
        /* ── Compact layout: single row (original) ── */
        // eslint-disable-next-line jsx-a11y/no-static-element-interactions -- mousedown prevention is intentional to preserve terminal focus
        <div
          className="flex items-center gap-0.5 px-1 py-1 touch-manipulation select-none"
          onMouseDown={(e) => e.preventDefault()}
        >
          {/* Combo picker + Modifier toggle buttons */}
          <KeyComboPicker onInput={onInput} onFocusTerminal={onFocusTerminal} />
          <div className="w-px h-4 bg-wc-default shrink-0" />
          {(["ctrl", "alt", "shift"] as const).map((mod) => (
            <button
              key={mod}
              data-testid={`toolbar-mod-${mod}`}
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={() => toggleModifier(mod)}
              className={cn(
                "shrink-0 rounded border px-1.5 py-1 text-xs font-medium transition touch-manipulation",
                modifiers[mod]
                  ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                  : "border-wc-default bg-wc-surface-input text-wc-text-secondary active:bg-wc-accent-active",
              )}
            >
              {mod.charAt(0).toUpperCase() + mod.slice(1)}
            </button>
          ))}
          <div className="w-px h-4 bg-wc-default shrink-0" />
          <div className="flex items-center gap-0.5 overflow-x-auto min-w-0 flex-1">
            {TOOLBAR_KEYS.map((key) => (
              <button
                key={key.label}
                data-testid={`toolbar-key-${slugify(key.label)}`}
                tabIndex={-1}
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
          {onUploadImage && (
            <button
              data-testid="toolbar-upload-image"
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={onUploadImage}
              className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
              title="Upload image"
            >
              <Image className="h-3.5 w-3.5" />
            </button>
          )}
          {voiceSupported && onVoiceStart && onVoiceStop && (
            <VoiceMicButton
              supported={voiceSupported}
              isRecording={voiceRecording ?? false}
              isTranscribing={voiceTranscribing ?? false}
              error={voiceError ?? null}
              audioLevel={voiceLevel}
              partialTranscript={voicePartialTranscript}
              backend={voiceBackend}
              onStart={onVoiceStart}
              onStop={onVoiceStop}
            />
          )}
        </div>
      )}
    </div>
  );
});
