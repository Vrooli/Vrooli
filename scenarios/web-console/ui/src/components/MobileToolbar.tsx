// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useDeferredValue, useRef, useState, useEffect, forwardRef, useImperativeHandle } from "react";
import { Image, SendHorizontal, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { TOOLBAR_KEYS, ESC_KEY, TAB_KEY, ENTER_KEY, ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, type ToolbarKey, applyModifiers } from "../consts/toolbar-keys";
import { strings } from "../consts/strings";
import type { GateResult, InputSource } from "./terminal/inputGate";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import KeyComboPicker from "./KeyComboPicker";
import VoiceMicButton from "./VoiceMicButton";
import VoiceCommandSuggestion from "./VoiceCommandSuggestion";
import AiSuggestBar from "./AiSuggestBar";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../hooks/useVoiceInput";
import type { CommandSuggestion } from "../domains/audio";
import { slugify } from "../lib/slugify";
import { useDraftPersistence } from "../hooks/useDraftPersistence";
import { useHoldRepeat } from "../hooks/useHoldRepeat";

/**
 * Arrow keys are the only toolbar buttons with hold-to-repeat because users
 * routinely need to scan through shell history, long command lines, and TUI
 * views. Other toolbar buttons (Esc/Tab/Enter, modifiers) are one-shot
 * actions where repeat would only cause accidental misfires.
 */
const ARROW_KEYS = new Set<string>([ARROW_UP.input, ARROW_DOWN.input, ARROW_LEFT.input, ARROW_RIGHT.input]);

interface ArrowToolbarButtonProps {
  keyDef: ToolbarKey;
  onFire: (key: ToolbarKey) => void;
  className: string;
}

/**
 * A toolbar arrow button that fires on pointerdown and auto-repeats while
 * held (via useHoldRepeat). Intentionally does NOT bind onClick — pointerdown
 * already dispatches, and a parallel click handler would double-fire on tap.
 */
function ArrowToolbarButton({ keyDef, onFire, className }: ArrowToolbarButtonProps) {
  const handlers = useHoldRepeat({ onFire: useCallback(() => onFire(keyDef), [onFire, keyDef]) });
  return (
    <button
      data-testid={`toolbar-key-${slugify(keyDef.label)}`}
      tabIndex={-1}
      {...handlers}
      className={className}
    >
      {keyDef.label}
    </button>
  );
}

// [REQ:P0-007a] Floating Toolbar Component
// [REQ:P0-007b] Terminal Key/Chord Mapping

/** Max visible lines before the textarea stops growing. */
const MAX_VISIBLE_LINES = 4;
/** Approximate line height in px for the textarea. */
const LINE_HEIGHT_PX = 20;
/** Max textarea height: MAX_VISIBLE_LINES * line-height + padding. */
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;

type SendStatus = "sent" | "queued" | "sending" | "failed" | "idle";

/** Snapshot of a single unsent payload used by the pending-input pill. */
export interface PendingInputSnapshot {
  data: string;
  addedAt: number;
}

export interface MobileToolbarHandle {
  /** Append text to the command input (used by voice transcription on mobile). */
  appendText: (text: string) => void;
  /** Focus the command textarea. Used after mic permission is granted so the
   *  user lands in a useful input target instead of nowhere. */
  focusInput: () => void;
  /** Clear the command textarea and its draft persistence. */
  clearInput: () => void;
}

interface MobileToolbarProps {
  /**
   * Callback to inject input into the active terminal via the input
   * gate. Returns a typed GateResult: `sent` (queued stdin_ack),
   * `queued` (session not ready, ws closed, or paused by xterm mode),
   * or `rejected` (empty or disposed). The toolbar uses the result
   * to surface queued/paused states as distinct pill variants.
   */
  onInput: (data: string, source: InputSource) => GateResult;
  /**
   * Subscribe to per-send settlement callbacks from the active terminal
   * socket. The draft is preserved during "sending" state and only cleared
   * after `ok === true` arrives; on `ok === false` the toolbar surfaces
   * "Send failed — retry" and restores the draft for editing.
   */
  subscribeInputSettled?: (cb: (seq: number, ok: boolean) => void) => () => void;
  /** Subscribe to pending-queue-changed notifications for the unsent pill. */
  subscribePendingInput?: (cb: () => void) => () => void;
  /** Snapshot the active terminal's pending (unsent) input queue. */
  getPendingInputSnapshot?: () => readonly PendingInputSnapshot[];
  /** Move focus to the active terminal (e.g. after submitting a command). */
  onFocusTerminal?: () => void;
  /** Active session ID for per-tab draft persistence. */
  activeSessionId?: string | null;
  /** Whether the toolbar is visible. */
  visible?: boolean;
  // Voice input props (optional - hidden when undefined)
  voiceSupported?: boolean;
  voicePreparing?: boolean;
  voiceRecording?: boolean;
  /** True when persistent voice mode is active. */
  voiceListening?: boolean;
  voiceTranscribing?: boolean;
  voiceError?: string | null;
  /** 0–1 audio level for live mic visualization */
  voiceLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  voicePartialTranscript?: string;
  voiceBackend?: string;
  onVoiceStart?: (opts?: StartRecordingOpts) => void;
  onVoiceStop?: () => void;
  onVoiceCancel?: () => void;
  /** Current voice command suggestion awaiting confirmation. */
  voiceCommandSuggestion?: CommandSuggestion | null;
  /** Called when user confirms a voice command suggestion. */
  onVoiceCommandConfirm?: (suggestion: CommandSuggestion) => void;
  /** Called when user dismisses a voice command suggestion. */
  onVoiceCommandDismiss?: (suggestion: CommandSuggestion) => void;
  onUploadImage?: () => void;
  /** Open the AI Command modal. Moved here from the floating toolbar on
   *  mobile because it's more accessible in the persistent bottom bar. */
  onOpenAi?: () => void;
  /** Whether the inline AI suggestion bar is active. Highlights the sparkles button. */
  aiSuggestActive?: boolean;
  /** Execute a suggestion generated from the local input draft. */
  onAiSuggestExecute?: (command: string) => void;
  /** Whether TTS is currently playing audio on the active pane. */
  isTtsSpeaking?: boolean;
  /** Stop TTS playback. */
  onTtsStop?: () => void;
  /** Current view mode of the active pane. Terminal-specific keys are hidden in messages mode. */
  viewMode?: "terminal" | "messages";
  /** Auto-switch to terminal view after sending a command while in messages mode. */
  onSwitchToTerminal?: () => void;
}

export default forwardRef<MobileToolbarHandle, MobileToolbarProps>(function MobileToolbar({
  onInput,
  subscribeInputSettled,
  subscribePendingInput,
  getPendingInputSnapshot,
  onFocusTerminal,
  activeSessionId,
  visible = true,
  voiceSupported,
  voicePreparing,
  voiceRecording,
  voiceListening,
  voiceTranscribing,
  voiceError,
  voiceLevel,
  voiceActivity,
  voicePartialTranscript,
  voiceBackend,
  onVoiceStart,
  onVoiceStop,
  onVoiceCancel,
  voiceCommandSuggestion,
  onVoiceCommandConfirm,
  onVoiceCommandDismiss,
  onUploadImage,
  onOpenAi,
  aiSuggestActive,
  onAiSuggestExecute,
  isTtsSpeaking,
  onTtsStop,
  viewMode = "terminal",
  onSwitchToTerminal,
}, ref) {
  const { t } = useTranslation();
  const { value: inputValue, setValue: setInputValue, clearDraft } = useDraftPersistence(activeSessionId);
  const deferredInputValue = useDeferredValue(inputValue);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  /**
   * Last-known caret position in the textarea. Tracked on select/blur so that
   * voice transcripts can be inserted at the user's caret even when focus has
   * moved to the mic button. `null` means we don't have a reliable position
   * (e.g. textarea has never been focused) — callers fall back to end-of-text.
   */
  const selectionRef = useRef<{ start: number; end: number } | null>(null);

  useImperativeHandle(ref, () => ({
    appendText: (text: string) => {
      const el = textareaRef.current;
      let insertEnd = 0;
      setInputValue(prev => {
        // Prefer live selection if textarea is focused; otherwise fall back
        // to the last-known caret position; otherwise append to end.
        let start = prev.length;
        let end = prev.length;
        if (el && document.activeElement === el && el.selectionStart !== null && el.selectionEnd !== null) {
          start = Math.min(el.selectionStart, prev.length);
          end = Math.min(el.selectionEnd, prev.length);
        } else if (selectionRef.current) {
          start = Math.min(selectionRef.current.start, prev.length);
          end = Math.min(selectionRef.current.end, prev.length);
        }
        const before = prev.slice(0, start);
        const after = prev.slice(end);
        const lead = before.length > 0 && !/\s$/.test(before) && !/^\s/.test(text) ? " " : "";
        const trail = after.length > 0 && !/^\s/.test(after) && !/\s$/.test(text) ? " " : "";
        const insertion = lead + text + trail;
        insertEnd = before.length + lead.length + text.length;
        return before + insertion + after;
      });
      // Restore caret to the end of the inserted text after React re-renders.
      requestAnimationFrame(() => {
        const node = textareaRef.current;
        if (!node) return;
        try {
          node.setSelectionRange(insertEnd, insertEnd);
        } catch {
          // setSelectionRange can throw if the element is detached; ignore.
        }
        selectionRef.current = { start: insertEnd, end: insertEnd };
      });
    },
    focusInput: () => {
      textareaRef.current?.focus();
    },
    clearInput: () => {
      clearDraft();
      selectionRef.current = null;
    },
  }), [setInputValue, clearDraft]);
  const [sendStatus, setSendStatus] = useState<SendStatus>("idle");
  /** Draft snapshot taken at submit time; restored on ack failure. */
  const pendingSendRef = useRef<{ draft: string } | null>(null);
  /** Unsubscribe for the current in-flight settlement subscription. */
  const settlementUnsubRef = useRef<(() => void) | null>(null);
  const [pendingInputEntries, setPendingInputEntries] = useState<readonly PendingInputSnapshot[]>([]);
  const [pillOpen, setPillOpen] = useState(false);
  const toolbarLayout = useWorkspaceStore((s) => s.toolbarLayout);
  const modifiers = useWorkspaceStore((s) => s.modifiers);
  const toggleModifier = useWorkspaceStore((s) => s.toggleModifier);
  const clearModifiers = useWorkspaceStore((s) => s.clearModifiers);
  const statusTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clear status timer on unmount
  useEffect(() => {
    return () => {
      if (statusTimerRef.current !== null) clearTimeout(statusTimerRef.current);
      settlementUnsubRef.current?.();
      settlementUnsubRef.current = null;
    };
  }, []);

  // Keep the pending-input pill in sync with the active terminal's queue.
  useEffect(() => {
    if (!subscribePendingInput || !getPendingInputSnapshot) {
      setPendingInputEntries([]);
      return;
    }
    const sync = () => setPendingInputEntries(getPendingInputSnapshot());
    sync();
    const unsub = subscribePendingInput(sync);
    return () => {
      unsub();
    };
  }, [subscribePendingInput, getPendingInputSnapshot]);

  // Auto-resize textarea height. We reset to "auto" first so scrollHeight
  // reflects the natural content height (otherwise it stays pinned at the
  // previous size). Skip the second write when the height hasn't changed to
  // avoid a redundant layout pass on every keystroke — that was contributing
  // to the laggy typing feel on mobile.
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    const target = Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT);
    const next = `${target}px`;
    if (el.style.height !== next) el.style.height = next;
  }, [inputValue]);

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
      onInput(data, "toolbar-key");
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

  /**
   * Subscribe to the next single settlement event from the terminal socket.
   * The subscription auto-unsubscribes after it fires once — subsequent
   * acks from other senders (e.g. xterm direct keystrokes) are ignored.
   */
  const awaitNextSettlement = useCallback(
    (onSettle: (ok: boolean) => void) => {
      if (!subscribeInputSettled) return;
      settlementUnsubRef.current?.();
      const unsub = subscribeInputSettled((_seq, ok) => {
        settlementUnsubRef.current?.();
        settlementUnsubRef.current = null;
        onSettle(ok);
      });
      settlementUnsubRef.current = unsub;
    },
    [subscribeInputSettled],
  );

  const submitCommand = useCallback(() => {
    // When the text box is exactly empty (length 0, NOT whitespace-only),
    // act as an Enter key press. This lets mobile users tap Send twice to
    // type a command and then confirm it with Enter — a very common pattern
    // when using interactive CLI tools like Claude Code. Whitespace-only
    // input is intentionally NOT treated as empty so it can still be
    // submitted verbatim (some programs interpret whitespace input).
    if (inputValue.length === 0) {
      onInput(ENTER_KEY.input, "toolbar-key");
      // Auto-switch to terminal so the user sees the result of pressing Enter
      if (viewMode === "messages") onSwitchToTerminal?.();
      onFocusTerminal?.();
      return;
    }

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

    // Snapshot the draft so we can restore it on ack failure. The draft
    // is kept visible during "sending" state; the ack resolution path
    // (below) decides whether to clear it.
    pendingSendRef.current = { draft: inputValue };

    const result = onInput(dataToSend, "toolbar-submit");

    if (result.status === "rejected") {
      // "empty" cannot occur here (inputValue.length > 0 checked above);
      // "disposed" means the pane has torn down. In either case the
      // draft should stay visible and no status change is needed.
      pendingSendRef.current = null;
      return;
    }

    if (result.status === "queued") {
      // The input was not sent immediately. Reason tells us why:
      //   - "not-ready"  — session_ready hasn't arrived; flushes later.
      //   - "ws-closed"  — socket is reconnecting; flushes on next open.
      //   - "paused"     — gate held it back (mouse-tracking mode etc.).
      // Preserve the draft (user sees the pending-input pill).
      showStatus("queued");
      onFocusTerminal?.();
      return;
    }

    // Frame was handed to the WS stack. Wait for stdin_ack before clearing.
    setSendStatus("sending");
    if (statusTimerRef.current !== null) {
      clearTimeout(statusTimerRef.current);
      statusTimerRef.current = null;
    }

    const finalizeSuccess = () => {
      pendingSendRef.current = null;
      clearDraft();
      showStatus("sent");
      if (viewMode === "messages") onSwitchToTerminal?.();
    };

    const finalizeFailure = () => {
      pendingSendRef.current = null;
      // Draft is still in the textarea (we never cleared it); just surface
      // the failure so the user can retry by pressing Send again.
      showStatus("failed");
    };

    if (subscribeInputSettled) {
      awaitNextSettlement((ok) => {
        if (ok) finalizeSuccess();
        else finalizeFailure();
      });
    } else {
      // No settlement seam available (legacy caller wiring) — fall back
      // to the optimistic behavior so we don't strand the draft.
      finalizeSuccess();
    }

    // After submitting a command, focus the terminal so the user can
    // immediately see and interact with the output.
    onFocusTerminal?.();
  }, [inputValue, onInput, clearDraft, showStatus, onFocusTerminal, clearModifiers, viewMode, onSwitchToTerminal, subscribeInputSettled, awaitNextSettlement]);

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      // pb-[var(--wc-safe-bottom)] adds bottom padding equal to the device's
      // safe-area inset (for rounded corners / home indicators in PWA mode).
      // The useAppViewport hook sets --wc-safe-bottom to 0px when the virtual
      // keyboard is open since the keyboard covers the bottom edge.
      className="flex shrink-0 flex-col border-t border-wc-default bg-wc-surface-raised touch-manipulation pb-[var(--wc-safe-bottom)] ps-[max(0.25rem,var(--wc-safe-left,0px))] pe-[max(0.25rem,var(--wc-safe-right,0px))]"
    >
      {/* Pending-input pill — visible whenever the terminal's stdin queue is non-empty.
          Clicking it toggles a disclosure listing truncated payloads and oldest age. */}
      {pendingInputEntries.length > 0 && (
        <div
          data-testid="pending-input-pill"
          className="flex flex-col border-b border-wc-default bg-wc-surface-raised/80 px-2 py-1 text-[11px] text-yellow-300"
        >
          <button
            type="button"
            onPointerDown={(e) => e.preventDefault()}
            onClick={() => setPillOpen((v) => !v)}
            className="flex items-center justify-between gap-2 text-start"
            title={t(strings.mobileToolbar.showUnsentTitle)}
          >
            <span>
              ⏳ {t(strings.mobileToolbar.unsentCount, { count: pendingInputEntries.length })}
              {(() => {
                const oldest = pendingInputEntries.reduce(
                  (min, e) => (min === null || e.addedAt < min ? e.addedAt : min),
                  null as number | null,
                );
                if (oldest === null) return null;
                const ageSec = Math.max(0, Math.floor((Date.now() - oldest) / 1000));
                return <span className="ms-1 text-wc-text-muted">{t(strings.mobileToolbar.unsentOldest, { seconds: ageSec })}</span>;
              })()}
            </span>
            <span className="text-wc-text-muted">{pillOpen ? "▾" : "▸"}</span>
          </button>
          {pillOpen && (
            <div data-testid="pending-input-disclosure" className="mt-1 flex flex-col gap-1">
              <ul className="max-h-24 overflow-y-auto font-mono text-[10px] text-wc-text-secondary">
                {pendingInputEntries.map((entry, idx) => {
                  const truncated = entry.data.length > 60 ? entry.data.slice(0, 60) + "…" : entry.data;
                  const ageSec = Math.max(0, Math.floor((Date.now() - entry.addedAt) / 1000));
                  return (
                    <li key={idx} className="truncate">
                      <span className="text-wc-text-muted">[{ageSec}s]</span> {truncated.replace(/\n/g, "⏎")}
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
        </div>
      )}
      {/* Voice command suggestion bar */}
      {voiceCommandSuggestion && onVoiceCommandConfirm && onVoiceCommandDismiss && (
        <VoiceCommandSuggestion
          suggestion={voiceCommandSuggestion}
          onConfirm={onVoiceCommandConfirm}
          onDismiss={onVoiceCommandDismiss}
        />
      )}
      {aiSuggestActive && onAiSuggestExecute && (
        <AiSuggestBar
          inputText={deferredInputValue}
          onExecute={onAiSuggestExecute}
          onClose={() => onOpenAi?.()}
        />
      )}
      {/* Command input row */}
      <div className="flex items-end gap-0.5 px-1 py-1">
        <div className="flex min-w-0 flex-1 flex-col">
          <textarea
            ref={textareaRef}
            data-testid="mobile-command-input"
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              selectionRef.current = {
                start: e.target.selectionStart ?? e.target.value.length,
                end: e.target.selectionEnd ?? e.target.value.length,
              };
            }}
            onSelect={(e) => {
              const t = e.currentTarget;
              selectionRef.current = {
                start: t.selectionStart ?? t.value.length,
                end: t.selectionEnd ?? t.value.length,
              };
            }}
            onBlur={(e) => {
              const t = e.currentTarget;
              selectionRef.current = {
                start: t.selectionStart ?? t.value.length,
                end: t.selectionEnd ?? t.value.length,
              };
            }}
            autoComplete="off"
            autoCorrect="on"
            spellCheck={false}
            rows={1}
            placeholder={t(strings.mobileToolbar.placeholder)}
            className={cn(
              "min-w-0 resize-none rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent",
              "overflow-y-auto overflow-x-hidden",
            )}
            style={{
              lineHeight: `${LINE_HEIGHT_PX}px`,
              maxHeight: `${MAX_TEXTAREA_HEIGHT}px`,
            }}
          />
          {sendStatus === "queued" && (
            <span data-testid="send-status-queued" className="px-1 text-[10px] text-yellow-400">
              {t(strings.mobileToolbar.statusQueued)}
            </span>
          )}
          {sendStatus === "sending" && (
            <span data-testid="send-status-sending" className="px-1 text-[10px] text-wc-text-muted">
              {t(strings.mobileToolbar.statusSending)}
            </span>
          )}
          {sendStatus === "failed" && (
            <span data-testid="send-status-failed" className="px-1 text-[10px] text-red-400">
              {t(strings.mobileToolbar.statusFailed)}
            </span>
          )}
        </div>
        {/* Send button — always enabled so that tapping it with an empty
             input acts as Enter (see submitCommand for rationale). */}
        <button
          data-testid="mobile-command-submit"
          onPointerDown={(e) => e.preventDefault()}
          onClick={submitCommand}
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
          title={t(strings.mobileToolbar.sendTitle)}
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
      {viewMode === "messages" ? (
        /* ── Messages mode: AI + image upload + voice mic ── */
        <div
          data-testid="messages-toolbar-actions"
          className="flex items-stretch gap-0.5 px-1 py-1 touch-manipulation select-none"
          onMouseDown={(e) => e.preventDefault()}
        >
          {onOpenAi && (
            <button
              data-testid="toolbar-ai"
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={onOpenAi}
              className={cn(
                "flex min-w-0 flex-1 items-center justify-center rounded border p-1.5 transition active:bg-wc-accent-active touch-manipulation",
                aiSuggestActive
                  ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                  : "border-wc-default bg-wc-surface-input text-wc-text-secondary",
              )}
              title={t(strings.mobileToolbar.aiCommandTitle)}
            >
              <Sparkles className="h-3.5 w-3.5" />
            </button>
          )}
          {onUploadImage && (
            <button
              data-testid="toolbar-upload-image"
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={onUploadImage}
              className="flex min-w-0 flex-1 items-center justify-center rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
              title={t(strings.mobileToolbar.uploadImageTitle)}
            >
              <Image className="h-3.5 w-3.5" />
            </button>
          )}
          {voiceSupported && onVoiceStart && onVoiceStop && (
            <VoiceMicButton
              supported={voiceSupported}
              isPreparing={voicePreparing ?? false}
              isRecording={voiceRecording ?? false}
              isListening={voiceListening ?? false}
              isTranscribing={voiceTranscribing ?? false}
              error={voiceError ?? null}
              audioLevel={voiceLevel}
              voiceActivity={voiceActivity}
              partialTranscript={voicePartialTranscript}
              backend={voiceBackend}
              isTtsSpeaking={isTtsSpeaking}
              onStart={onVoiceStart}
              onStop={onVoiceStop}
              onCancel={onVoiceCancel}
              onTtsStop={onTtsStop}
              className="flex min-w-0 flex-1"
              buttonClassName="flex w-full items-center justify-center"
            />
          )}
        </div>
      ) : toolbarLayout === "expanded" ? (
        /* ── Expanded layout: two rows with D-pad arrow cluster ──
           ┌────────────────────────────────────────────────────────────┐
           │ [Ctrl] [Alt] [Shift] │     [↑]      │ [📷] │            │
           │ [Esc]  [Tab] [Enter] │ [←] [↓] [→]  │      │    [🎤]   │
           └────────────────────────────────────────────────────────────┘
           The mic button spans both rows for easy access. */
        <div
          className="grid gap-0.5 px-1 py-1 touch-manipulation select-none"
          // Mic column is `minmax(0,1fr)` so it grows to fill remaining width
          // (up to ~2× the image-upload button) but is allowed to shrink when
          // the other columns are wide — preventing viewport overflow.
          style={{ gridTemplateColumns: "auto auto auto minmax(0,1fr)", gridTemplateRows: "auto auto" }}
          onMouseDown={(e) => e.preventDefault()}
        >
          {/* Column 1: Combo picker + Modifiers (row 1) + Special keys (row 2) */}
          <div className="flex flex-col gap-0.5" style={{ gridRow: "1 / -1" }}>
            <div className="flex items-stretch gap-0.5">
              <KeyComboPicker
                onInput={onInput}
                onFocusTerminal={onFocusTerminal}
                triggerClassName="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation flex items-center justify-center"
              />
              <div className="w-px self-stretch bg-wc-default shrink-0" />
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

          {/* Column 2: D-pad arrow cluster (hold-to-repeat via ArrowToolbarButton) */}
          <div className="flex flex-col items-center gap-0.5 px-1" style={{ gridRow: "1 / -1" }}>
            {/* Row 1: Up arrow centered */}
            <div className="flex justify-center">
              <ArrowToolbarButton
                keyDef={ARROW_UP}
                onFire={handleKey}
                className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]"
              />
            </div>
            {/* Row 2: Left, Down, Right */}
            <div className="flex items-center gap-0.5">
              {[ARROW_LEFT, ARROW_DOWN, ARROW_RIGHT].map((key) => (
                <ArrowToolbarButton
                  key={key.label}
                  keyDef={key}
                  onFire={handleKey}
                  className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]"
                />
              ))}
            </div>
          </div>

          {/* Column 3: AI + Image upload buttons (right-aligned) */}
          <div className="flex flex-col items-end gap-0.5" style={{ gridColumn: 3, gridRow: "1 / -1" }}>
            {onOpenAi && (
              <button
                data-testid="toolbar-ai"
                tabIndex={-1}
                onPointerDown={(e) => e.preventDefault()}
                onClick={onOpenAi}
                className={cn(
                  "shrink-0 rounded border p-2 transition active:bg-wc-accent-active touch-manipulation",
                  aiSuggestActive
                    ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                    : "border-wc-default bg-wc-surface-input text-wc-text-secondary",
                )}
                title={t(strings.mobileToolbar.aiCommandTitle)}
              >
                <Sparkles className="h-4 w-4" />
              </button>
            )}
            {onUploadImage && (
              <button
                data-testid="toolbar-upload-image"
                tabIndex={-1}
                onPointerDown={(e) => e.preventDefault()}
                onClick={onUploadImage}
                className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-2 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
                title={t(strings.mobileToolbar.uploadImageTitle)}
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
                isPreparing={voicePreparing ?? false}
                isRecording={voiceRecording ?? false}
                isListening={voiceListening ?? false}
                isTranscribing={voiceTranscribing ?? false}
                error={voiceError ?? null}
                audioLevel={voiceLevel}
                voiceActivity={voiceActivity}
                partialTranscript={voicePartialTranscript}
                backend={voiceBackend}
                isTtsSpeaking={isTtsSpeaking}
                onStart={onVoiceStart}
                onStop={onVoiceStop}
                onCancel={onVoiceCancel}
                onTtsStop={onTtsStop}
                className="h-full w-full"
                // Mic stretches to fill the remaining row width (its grid
                // column is minmax(0,1fr)), making it as wide as fits without
                // pushing other buttons off-screen. min-width keeps it a
                // usable tap target even on very narrow viewports.
                buttonClassName="h-full w-full min-w-[2.5rem] flex items-center justify-center"
              />
            </div>
          )}
        </div>
      ) : (
        /* ── Compact layout: single row (original) ── */
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
            {TOOLBAR_KEYS.map((key) => {
              const className = cn(
                "shrink-0 rounded border border-wc-default bg-wc-surface-input px-1.5 py-1 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation",
                key.width === "wide" ? "min-w-[3.5rem]" : key.width === "narrow" ? "min-w-[1.75rem]" : "min-w-[2.25rem]",
              );
              if (ARROW_KEYS.has(key.input)) {
                return (
                  <ArrowToolbarButton
                    key={key.label}
                    keyDef={key}
                    onFire={handleKey}
                    className={className}
                  />
                );
              }
              return (
                <button
                  key={key.label}
                  data-testid={`toolbar-key-${slugify(key.label)}`}
                  tabIndex={-1}
                  onPointerDown={(e) => e.preventDefault()}
                  onClick={() => handleKey(key)}
                  className={className}
                >
                  {key.label}
                </button>
              );
            })}
          </div>
          {onOpenAi && (
            <button
              data-testid="toolbar-ai"
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={onOpenAi}
              className={cn(
                "shrink-0 rounded border p-1.5 transition active:bg-wc-accent-active touch-manipulation",
                aiSuggestActive
                  ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                  : "border-wc-default bg-wc-surface-input text-wc-text-secondary",
              )}
              title={t(strings.mobileToolbar.aiCommandTitle)}
            >
              <Sparkles className="h-3.5 w-3.5" />
            </button>
          )}
          {onUploadImage && (
            <button
              data-testid="toolbar-upload-image"
              tabIndex={-1}
              onPointerDown={(e) => e.preventDefault()}
              onClick={onUploadImage}
              className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
              title={t(strings.mobileToolbar.uploadImageTitle)}
            >
              <Image className="h-3.5 w-3.5" />
            </button>
          )}
          {voiceSupported && onVoiceStart && onVoiceStop && (
            <VoiceMicButton
              supported={voiceSupported}
              isPreparing={voicePreparing ?? false}
              isRecording={voiceRecording ?? false}
              isListening={voiceListening ?? false}
              isTranscribing={voiceTranscribing ?? false}
              error={voiceError ?? null}
              audioLevel={voiceLevel}
              voiceActivity={voiceActivity}
              partialTranscript={voicePartialTranscript}
              backend={voiceBackend}
              isTtsSpeaking={isTtsSpeaking}
              onStart={onVoiceStart}
              onStop={onVoiceStop}
              onCancel={onVoiceCancel}
              onTtsStop={onTtsStop}
            />
          )}
        </div>
      )}
    </div>
  );
});
