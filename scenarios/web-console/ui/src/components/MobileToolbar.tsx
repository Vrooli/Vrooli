// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useDeferredValue, useRef, useState, useEffect, forwardRef, useImperativeHandle } from "react";
import { Image, Loader2, Maximize2, SendHorizontal, Sparkles } from "lucide-react";
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
import type { CommandSuggestion } from "../audio-integration";
import { slugify } from "../lib/slugify";
import { useComposerDraft, type ComposerDraft } from "../hooks/useComposerDraft";
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

/**
 * Wraps AiSuggestBar and applies useDeferredValue to the draft text *here*
 * rather than in MobileToolbar. Hoisting the deferral into a child that only
 * mounts while the suggest bar is open means a normal keystroke no longer
 * triggers MobileToolbar's deferred (second) render pass when the bar is
 * closed — which is the common case and was adding latency to typing.
 */
function DeferredAiSuggestBar({
  inputText,
  onExecute,
  onClose,
}: {
  inputText: string;
  onExecute: (command: string) => void;
  onClose: () => void;
}) {
  const deferredInputText = useDeferredValue(inputText);
  return <AiSuggestBar inputText={deferredInputText} onExecute={onExecute} onClose={onClose} />;
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
  /**
   * Shared per-session draft. When provided (by Workspace), the collapsed input
   * and the full-screen composer read/write ONE value and cannot diverge. When
   * omitted (standalone/tests), the toolbar owns a private draft instead.
   */
  draft?: ComposerDraft;
  /**
   * Open the full-screen composer for the active session. When provided, a
   * low-emphasis corner expand icon is shown on the textarea (terminal mode).
   */
  onExpandComposer?: () => void;
  /** Whether the toolbar is visible. */
  visible?: boolean;
  // Voice input props (optional - hidden when undefined)
  voiceSupported?: boolean;
  voicePreparing?: boolean;
  voiceRecording?: boolean;
  /** True when persistent voice mode is active. */
  voiceListening?: boolean;
  /** True when passive wake-word listening currently holds the mic. */
  voicePassive?: boolean;
  voiceTranscribing?: boolean;
  /** True when a live mic lease is orphaned while the UI is idle (recovery). */
  voiceStaleLiveMic?: boolean;
  voiceError?: string | null;
  /** 0–1 audio level for live mic visualization */
  voiceLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  voicePartialTranscript?: string;
  voiceBackend?: string;
  voiceCanExportDiagnostic?: boolean;
  onVoiceExportDiagnostic?: () => string | null;
  onVoiceStart?: (opts?: StartRecordingOpts) => void;
  onVoicePrepare?: () => void;
  onVoiceStop?: () => void;
  /** Exit passive wake-word listening (tapping the passive mic button). */
  onVoiceExitPassive?: () => void;
  /** Release an orphaned live mic lease (stale-live-mic recovery). */
  onVoiceReleaseMic?: () => void;
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
  draft: draftProp,
  onExpandComposer,
  visible = true,
  voiceSupported,
  voicePreparing,
  voiceRecording,
  voiceListening,
  voicePassive,
  voiceTranscribing,
  voiceStaleLiveMic,
  voiceError,
  voiceLevel,
  voiceActivity,
  voicePartialTranscript,
  voiceBackend,
  voiceCanExportDiagnostic,
  onVoiceExportDiagnostic,
  onVoiceStart,
  onVoicePrepare,
  onVoiceStop,
  onVoiceExitPassive,
  onVoiceReleaseMic,
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
  // Single-source the draft. When Workspace passes a shared draft, the collapsed
  // input and the full-screen composer read/write the same value; the private
  // fallback keeps the toolbar usable standalone (and in unit tests). Both hooks
  // run unconditionally (hooks rule) but only the selected one is wired up.
  const fallbackDraft = useComposerDraft(activeSessionId);
  const draft = draftProp ?? fallbackDraft;
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // The textarea is UNCONTROLLED: the DOM owns the live value and the shared
  // draft mirrors it. Typing therefore never calls setState, so a keystroke
  // doesn't re-render the toolbar — that controlled round-trip was the main
  // cause of mobile typing lag. (Backspace is likewise left entirely to the
  // browser, which natively handles tap-to-delete and hold-to-repeat in any
  // textarea.) React state is used only for things that must re-render: the
  // AI-suggest draft (below) and send status.

  // The AI-suggest bar needs the live draft to render suggestions, which means
  // re-rendering on each keystroke — but ONLY while it's open. We mirror the
  // draft into state guarded by an `aiSuggestActive` ref so normal typing (bar
  // closed, the common case) stays state-free and lag-free.
  const [aiInputText, setAiInputText] = useState(() => draft.getValue());
  const aiSuggestActiveRef = useRef(false);
  useEffect(() => {
    aiSuggestActiveRef.current = !!aiSuggestActive;
    if (aiSuggestActive) setAiInputText(draft.getValue());
  }, [aiSuggestActive, draft]);

  // Auto-resize the textarea to fit its content (up to MAX_TEXTAREA_HEIGHT).
  // Reset to "auto" first so scrollHeight reflects the natural content height,
  // then only write back when it actually changed to avoid a redundant layout.
  const resizeTextarea = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    const target = Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT);
    const next = `${target}px`;
    if (el.style.height !== next) el.style.height = next;
  }, []);

  // Track the live value + caret as the user types. No setState in the common
  // path → no re-render per keystroke. The shared draft notifies peer surfaces.
  const handleTextareaChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    draft.handleChange(e.currentTarget);
    resizeTextarea();
  }, [draft, resizeTextarea]);

  // Size the textarea to its initial draft once on mount.
  useEffect(() => {
    resizeTextarea();
  }, [resizeTextarea]);

  // Reseed the textarea from the shared draft whenever ANOTHER surface changes
  // it (the composer typing, voice insertion, clear, session reload). We skip
  // reseeding during our own typing (reason "input" while this textarea is
  // focused) so we never clobber the live caret. The AI-suggest mirror follows
  // the draft here so it stays state-free until the bar is open.
  useEffect(() => {
    return draft.subscribe((change) => {
      const el = textareaRef.current;
      const isOwnTyping = change.reason === "input" && el != null && document.activeElement === el;
      if (el && !isOwnTyping) {
        if (el.value !== change.value) el.value = change.value;
        if (change.caret != null) {
          try {
            el.setSelectionRange(change.caret, change.caret);
          } catch {
            // The textarea may be detached during teardown; ignore.
          }
        }
        resizeTextarea();
      }
      if (aiSuggestActiveRef.current) setAiInputText(change.value);
    });
  }, [draft, resizeTextarea]);

  useImperativeHandle(ref, () => ({
    appendText: (text: string) => {
      draft.appendAtCaret(text);
    },
    focusInput: () => {
      textareaRef.current?.focus();
    },
    clearInput: () => {
      draft.reset();
    },
  }), [draft]);
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
    const draftText = draft.getValue();
    if (draftText.length === 0) {
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
      const { data } = applyModifiers(draftText, mods);
      dataToSend = data;
      clearModifiers();
    } else {
      // Send exactly what the user typed — no appended newline.
      // The user can explicitly include a newline via the Enter toolbar key
      // if needed. Appending one automatically caused unwanted extra blank
      // lines in the terminal output.
      dataToSend = draftText;
    }

    // Snapshot the draft so we can restore it on ack failure. The draft
    // is kept visible during "sending" state; the ack resolution path
    // (below) decides whether to clear it.
    pendingSendRef.current = { draft: draftText };

    const result = onInput(dataToSend, "toolbar-submit");

    if (result.status === "rejected") {
      // "empty" cannot occur here (draft.length > 0 checked above);
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
      draft.reset();
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
  }, [onInput, draft, showStatus, onFocusTerminal, clearModifiers, viewMode, onSwitchToTerminal, subscribeInputSettled, awaitNextSettlement]);

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      // Do not add bottom safe-area padding here. The toolbar is part of the
      // fixed app layout, and iOS/PWA safe-area handling already reserves the
      // bottom edge; adding it here creates a visible extra gutter.
      className="wc-chrome-surface-raised flex shrink-0 flex-col border-t border-wc-default touch-manipulation ps-[max(0.25rem,var(--wc-safe-left,0px))] pe-[max(0.25rem,var(--wc-safe-right,0px))]"
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
        <DeferredAiSuggestBar
          inputText={aiInputText}
          onExecute={onAiSuggestExecute}
          onClose={() => onOpenAi?.()}
        />
      )}
      {/* Command input row */}
      <div className="flex items-end gap-0.5 px-1 py-1">
        <div className="flex min-w-0 flex-1 flex-col">
          {/* Uncontrolled: the DOM owns the value (mirrored into the shared
              draft), so typing and native backspace never re-render the toolbar. */}
          <div className="relative flex min-w-0">
            <textarea
              ref={textareaRef}
              data-testid="mobile-command-input"
              defaultValue={draft.getValue()}
              onChange={handleTextareaChange}
              onSelect={(e) => draft.trackSelection(e.currentTarget)}
              onBlur={(e) => draft.trackSelection(e.currentTarget)}
              autoComplete="off"
              autoCorrect="on"
              spellCheck={false}
              rows={1}
              placeholder={t(strings.mobileToolbar.placeholder)}
              className={cn(
                "min-w-0 flex-1 resize-none rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent",
                "overflow-y-auto overflow-x-hidden",
                // Reserve room for the corner expand icon so long text never
                // slides under it.
                onExpandComposer && "pe-7",
              )}
              style={{
                lineHeight: `${LINE_HEIGHT_PX}px`,
                maxHeight: `${MAX_TEXTAREA_HEIGHT}px`,
              }}
            />
            {/* Always-visible, low-emphasis corner icon that expands the draft
                into the full-screen composer. Keyboard-reachable and shown in
                both terminal and messages views — the composer is view-agnostic
                and the long-message pain applies equally when chatting. */}
            {onExpandComposer && (
              <button
                type="button"
                data-testid="expand-toggle"
                onPointerDown={(e) => e.preventDefault()}
                onClick={onExpandComposer}
                className="absolute end-1 top-1 rounded p-0.5 text-wc-text-muted/70 transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
                title={t(strings.mobileToolbar.expandComposerTitle)}
                aria-label={t(strings.mobileToolbar.expandComposerTitle)}
              >
                <Maximize2 className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          {sendStatus === "queued" && (
            <span data-testid="send-status-queued" className="px-1 text-[10px] text-yellow-400">
              {t(strings.mobileToolbar.statusQueued)}
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
          title={sendStatus === "sending" ? t(strings.mobileToolbar.statusSending) : t(strings.mobileToolbar.sendTitle)}
        >
          {/* "sending" renders as an inline spinner in the button itself rather
              than a label below the textarea — the label changed the toolbar's
              height on every send, forcing a costly terminal resize/reflow. */}
          {sendStatus === "sending" ? (
            <Loader2 data-testid="send-status-sending" className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <SendHorizontal className="h-3.5 w-3.5" />
          )}
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
              isPassive={voicePassive ?? false}
              isTranscribing={voiceTranscribing ?? false}
              staleLiveMic={voiceStaleLiveMic ?? false}
              error={voiceError ?? null}
              audioLevel={voiceLevel}
              voiceActivity={voiceActivity}
              partialTranscript={voicePartialTranscript}
              backend={voiceBackend}
              canExportDiagnostic={voiceCanExportDiagnostic}
              onExportDiagnostic={onVoiceExportDiagnostic}
              isTtsSpeaking={isTtsSpeaking}
              onPrepare={onVoicePrepare}
              onStart={onVoiceStart}
              onStop={onVoiceStop}
              onExitPassive={onVoiceExitPassive}
              onReleaseMic={onVoiceReleaseMic}
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
                staleLiveMic={voiceStaleLiveMic ?? false}
                error={voiceError ?? null}
                audioLevel={voiceLevel}
                voiceActivity={voiceActivity}
                partialTranscript={voicePartialTranscript}
                backend={voiceBackend}
                canExportDiagnostic={voiceCanExportDiagnostic}
                onExportDiagnostic={onVoiceExportDiagnostic}
                isTtsSpeaking={isTtsSpeaking}
                onPrepare={onVoicePrepare}
                onStart={onVoiceStart}
                onStop={onVoiceStop}
                onExitPassive={onVoiceExitPassive}
                onReleaseMic={onVoiceReleaseMic}
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
              isPassive={voicePassive ?? false}
              isTranscribing={voiceTranscribing ?? false}
              staleLiveMic={voiceStaleLiveMic ?? false}
              error={voiceError ?? null}
              audioLevel={voiceLevel}
              voiceActivity={voiceActivity}
              partialTranscript={voicePartialTranscript}
              backend={voiceBackend}
              canExportDiagnostic={voiceCanExportDiagnostic}
              onExportDiagnostic={onVoiceExportDiagnostic}
              isTtsSpeaking={isTtsSpeaking}
              onPrepare={onVoicePrepare}
              onStart={onVoiceStart}
              onStop={onVoiceStop}
              onExitPassive={onVoiceExitPassive}
              onReleaseMic={onVoiceReleaseMic}
              onCancel={onVoiceCancel}
              onTtsStop={onTtsStop}
            />
          )}
        </div>
      )}
    </div>
  );
});
