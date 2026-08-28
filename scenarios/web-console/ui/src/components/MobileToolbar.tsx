// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-key-combos-p0-007
import { useCallback, useDeferredValue, useMemo, useRef, useState, useEffect, forwardRef, useImperativeHandle, type ReactNode } from "react";
import { Loader2, Maximize2, SendHorizontal } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { useTranslation } from "react-i18next";
import { ENTER_KEY, type ToolbarKey, applyModifiers } from "../consts/toolbar-keys";
import { strings } from "../consts/strings";
import type { GateResult, InputIntent } from "./terminal/inputGate";
import type { InputSettlementCallback } from "../hooks/terminal/useStdinStream";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import KeyComboPicker from "./KeyComboPicker";
import VoiceCommandSuggestion from "./VoiceCommandSuggestion";
import InterimTranscriptOverlay from "./composer/InterimTranscriptOverlay";
import AiSuggestBar from "./AiSuggestBar";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../audio-integration";
import type { CommandSuggestion } from "../audio-integration";
import { useComposerDraft, type ComposerDraft } from "../hooks/useComposerDraft";
import { decodeInputLabel } from "../lib/terminalKeyLabels";
import {
  TOOLBAR_CONTROLS,
  layoutToolbar,
  toolbarMetrics,
  type ToolbarControlId,
  type ToolbarControlSpec,
} from "../lib/toolbarLayout";
import { useElementWidth } from "../hooks/useElementWidth";
import ToolbarSurface from "./toolbar/ToolbarSurface";
import { activeToolbarControlStyle, renderToolbarControl, type ToolbarControlContext } from "./toolbar/toolbarControls";

/**
 * Width assumed before the toolbar has been measured. In a browser the real
 * width lands in the same commit (ResizeObserver reports on observe), so this
 * only governs environments with no layout at all — SSR and jsdom — where a
 * mainstream phone width produces a representative toolbar instead of an empty
 * one.
 */
const INITIAL_TOOLBAR_WIDTH_PX = 390;

/** The More sheet always uses comfortable targets, whatever the toolbar does. */
const SHEET_METRICS = toolbarMetrics("large");

interface ToolbarIconButtonProps {
  testId: string;
  label: string;
  onClick: () => void;
  active?: boolean;
  className?: string;
  children: ReactNode;
}

/**
 * The mobile toolbar has several icon-only actions. Keep their hit target,
 * centering, colour mapping, and accessible naming on the shared RCL control
 * instead of rebuilding those details at each call site.
 */
function ToolbarIconButton({ testId, label, onClick, active = false, className, children }: ToolbarIconButtonProps) {
  return (
    <Button
      data-testid={testId}
      data-active={active ? "true" : undefined}
      aria-pressed={active}
      tabIndex={-1}
      type="button"
      aria-label={label}
      title={label}
      variant="secondary"
      size="sm"
      density="compact"
      onPointerDown={(e) => e.preventDefault()}
      onClick={onClick}
      className={cn(
        "flex min-h-11 min-w-11 items-center justify-center rounded border p-0 font-medium transition touch-manipulation",
        className,
      )}
      style={active ? activeToolbarControlStyle : undefined}
    >
      {children}
    </Button>
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
/** Max textarea height: MAX_VISIBLE_LINES * line-height + vertical padding. */
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 24;

type SendStatus = "sent" | "queued" | "sending" | "failed" | "idle";

/** Snapshot of a single unsent payload used by the pending-input pill. */
export interface PendingInputSnapshot {
  data: string;
  addedAt: number;
  intent: "typing" | "bulk_text" | "named_key";
  held?: boolean;
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

export interface MobileToolbarVoiceProps {
  supported?: boolean;
  preparing?: boolean;
  recording?: boolean;
  persistentMode?: boolean;
  listening?: boolean;
  passive?: boolean;
  transcribing?: boolean;
  error?: string | null;
  level?: number;
  activity?: VoiceActivitySnapshot;
  partialTranscript?: string;
  backend?: string;
  onStart?: (opts?: StartRecordingOpts) => void;
  onPrepare?: () => void;
  onStop?: () => void;
  onExitPassive?: () => void;
  commandSuggestion?: CommandSuggestion | null;
  onCommandConfirm?: (suggestion: CommandSuggestion) => void;
  onCommandDismiss?: (suggestion: CommandSuggestion) => void;
}

interface MobileToolbarProps {
  /**
   * Callback to inject input into the active terminal via the input
   * gate. Returns a typed GateResult: `sent` (queued stdin_ack),
   * `queued` (session not ready, ws closed, or paused by xterm mode),
   * or `rejected` (empty or disposed). The toolbar uses the result
   * to surface queued/paused states as distinct pill variants.
   */
  onInput: (data: string, intent: Exclude<InputIntent, "control">) => GateResult;
  /**
   * Subscribe to per-send settlement callbacks from the active terminal
   * socket. The draft is preserved during "sending" state and only cleared
   * after `ok === true` arrives; on `ok === false` the toolbar surfaces
   * "Send failed — retry" and restores the draft for editing.
   */
  subscribeInputSettled?: (cb: (offset: number, ok: boolean, reason?: string) => void) => () => void;
  /** Await settlement for the exact byte offset returned by onInput. */
  awaitOffset?: (offset: number, cb: InputSettlementCallback) => () => void;
  /** Subscribe to pending-queue-changed notifications for the unsent pill. */
  subscribePendingInput?: (cb: () => void) => () => void;
  /** Snapshot the active terminal's pending (unsent) input queue. */
  getPendingInputSnapshot?: () => readonly PendingInputSnapshot[];
  discardPendingInput?: (index: number) => void;
  discardAllPendingInput?: () => void;
  flushPendingInputNow?: () => void;
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
  /** Voice state and callbacks are grouped to keep the toolbar boundary small. */
  voice?: MobileToolbarVoiceProps;
  onUploadImage?: () => void;
  /** Open the AI Command modal. Moved here from the floating toolbar on
   *  mobile because it's more accessible in the persistent bottom bar. */
  onOpenAi?: () => void;
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
  awaitOffset,
  subscribePendingInput,
  getPendingInputSnapshot,
  discardPendingInput,
  discardAllPendingInput,
  flushPendingInputNow,
  onFocusTerminal,
  activeSessionId,
  draft: draftProp,
  onExpandComposer,
  visible = true,
  voice,
  onUploadImage,
  onOpenAi,
  onAiSuggestExecute,
  isTtsSpeaking,
  onTtsStop,
  viewMode = "terminal",
  onSwitchToTerminal,
}, ref) {
  const { t } = useTranslation();
  const aiSuggestActive = useWorkspaceStore((state) => state.aiSuggestActive);
  const {
    supported: voiceSupported,
    preparing: voicePreparing,
    recording: voiceRecording,
    persistentMode: voicePersistentMode,
    listening: voiceListening,
    passive: voicePassive,
    transcribing: voiceTranscribing,
    error: voiceError,
    level: voiceLevel,
    activity: voiceActivity,
    partialTranscript: voicePartialTranscript,
    backend: voiceBackend,
    onStart: onVoiceStart,
    onPrepare: onVoicePrepare,
    onStop: onVoiceStop,
    onExitPassive: onVoiceExitPassive,
    commandSuggestion: voiceCommandSuggestion,
    onCommandConfirm: onVoiceCommandConfirm,
    onCommandDismiss: onVoiceCommandDismiss,
  } = voice ?? {};
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
  const toolbarPrefs = useWorkspaceStore((s) => s.toolbarPrefs);
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
      onInput(data, "typing");
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
  const submitCommand = useCallback(() => {
    // When the text box is exactly empty (length 0, NOT whitespace-only),
    // act as an Enter key press. This lets mobile users tap Send twice to
    // type a command and then confirm it with Enter — a very common pattern
    // when using interactive CLI tools like Claude Code. Whitespace-only
    // input is intentionally NOT treated as empty so it can still be
    // submitted verbatim (some programs interpret whitespace input).
    const draftText = draft.getValue();
    if (draftText.length === 0) {
      onInput(ENTER_KEY.input, "typing");
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
    // Snapshot the session too: the ack can settle after the user switches
    // sessions, and the clear below must target the session we sent from.
    const sentFrom = draft.getSessionId();

    const result = onInput(dataToSend, "bulk_text");

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
      //   - "paused"     — an explicit higher-level pause held it back.
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
      draft.reset(sentFrom);
      showStatus("sent");
      if (viewMode === "messages") onSwitchToTerminal?.();
    };

    const finalizeFailure = () => {
      pendingSendRef.current = null;
      // Draft is still in the textarea (we never cleared it); just surface
      // the failure so the user can retry by pressing Send again.
      showStatus("failed");
    };

    if (awaitOffset) {
      settlementUnsubRef.current?.();
      settlementUnsubRef.current = awaitOffset(result.offset, (ok) => {
        settlementUnsubRef.current = null;
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
  }, [onInput, draft, showStatus, onFocusTerminal, clearModifiers, viewMode, onSwitchToTerminal, awaitOffset]);

  // ── Toolbar composition ────────────────────────────────────────────────
  // The arrangement is computed, not written. `layoutToolbar` decides which
  // controls fit the user's row budget at the width we actually have; the same
  // call with a simulated width drives the settings preview.
  const [keysAreaRef, measuredWidth] = useElementWidth();

  const voiceAvailable = Boolean(voiceSupported && onVoiceStart && onVoiceStop);

  /** Controls whose feature is not wired up this render claim no width. */
  const unavailable = useMemo<ToolbarControlId[]>(() => {
    const ids: ToolbarControlId[] = [];
    if (!onOpenAi) ids.push("ai");
    if (!onUploadImage) ids.push("image");
    if (!voiceAvailable) ids.push("mic");
    return ids;
  }, [onOpenAi, onUploadImage, voiceAvailable]);

  const layout = useMemo(
    () => layoutToolbar(toolbarPrefs, measuredWidth ?? INITIAL_TOOLBAR_WIDTH_PX, {
      view: viewMode,
      unavailable,
    }),
    [toolbarPrefs, measuredWidth, viewMode, unavailable],
  );

  const controlLabels = useMemo<Record<string, string>>(() => ({
    more: t(strings.mobileToolbar.controls.more),
    modifiers: t(strings.mobileToolbar.controls.modifiers),
    special: t(strings.mobileToolbar.controls.special),
    arrows: t(strings.mobileToolbar.controls.arrows),
    mic: t(strings.mobileToolbar.controls.mic),
    image: t(strings.mobileToolbar.uploadImageTitle),
    ai: t(strings.mobileToolbar.aiCommandTitle),
  }), [t]);

  const voiceProps = useMemo(() => (voiceAvailable && onVoiceStart && onVoiceStop ? {
    supported: true,
    isPreparing: voicePreparing ?? false,
    isRecording: voiceRecording ?? false,
    persistentMode: voicePersistentMode ?? false,
    isListening: voiceListening ?? false,
    isPassive: voicePassive ?? false,
    isTranscribing: voiceTranscribing ?? false,
    error: voiceError ?? null,
    audioLevel: voiceLevel,
    voiceActivity: voiceActivity,
    backend: voiceBackend,
    onPrepare: onVoicePrepare,
    onStart: onVoiceStart,
    onStop: onVoiceStop,
    onExitPassive: onVoiceExitPassive,
  } : undefined), [
    voiceAvailable, voicePreparing, voiceRecording, voicePersistentMode, voiceListening,
    voicePassive, voiceTranscribing, voiceError, voiceLevel, voiceActivity, voiceBackend,
    onVoicePrepare, onVoiceStart, onVoiceStop, onVoiceExitPassive,
  ]);

  /**
   * Controls that are not on the surface — hidden by the user, or pushed out by
   * the budget. They are the reason More is pinned: this list is what would
   * otherwise be unreachable.
   */
  const offToolbarSpecs = useMemo<ToolbarControlSpec[]>(() => {
    const seated = new Set<ToolbarControlId>(layout.rows.flatMap((row) => row.slots.map((s) => s.id)));
    if (layout.dpad) seated.add("arrows");
    return TOOLBAR_CONTROLS.filter((spec) => (
      !spec.pinned
      && !unavailable.includes(spec.id)
      && !(viewMode === "messages" && spec.terminalOnly)
      && !seated.has(spec.id)
    ));
  }, [layout, unavailable, viewMode]);

  const baseControlContext = useMemo<Omit<ToolbarControlContext, "moreTrigger">>(() => ({
    onKey: handleKey,
    modifiers,
    toggleModifier,
    onOpenAi,
    aiSuggestActive,
    onUploadImage,
    voice: voiceProps,
    labels: controlLabels,
  }), [handleKey, modifiers, toggleModifier, onOpenAi, aiSuggestActive, onUploadImage, voiceProps, controlLabels]);

  // Rendered inside the More sheet at comfortable targets. Same renderer as the
  // toolbar, so a control behaves identically wherever it is reached from.
  const offToolbarControls = useMemo(() => offToolbarSpecs.map((spec) => ({
    id: String(spec.id),
    label: controlLabels[spec.id] ?? String(spec.id),
    node: renderToolbarControl(
      { id: spec.id, spec, width: SHEET_METRICS.unit, fill: false },
      // A control can be in the strip and the sheet at once; the prefix keeps
      // the two copies distinguishable.
      { ...baseControlContext, testIdPrefix: "more-" },
      SHEET_METRICS,
    ),
  })), [offToolbarSpecs, controlLabels, baseControlContext]);

  const controlContext = useMemo<ToolbarControlContext>(() => ({
    ...baseControlContext,
    moreTrigger: ({ className, style, label }) => (
      <KeyComboPicker
        onInput={onInput}
        onFocusTerminal={onFocusTerminal}
        triggerClassName={className}
        triggerStyle={style}
        triggerLabel={label}
        offToolbarControls={offToolbarControls}
        showKeyCombos={viewMode === "terminal"}
      />
    ),
  }), [baseControlContext, onInput, onFocusTerminal, offToolbarControls, viewMode]);

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
                      <span className="text-wc-text-muted">[{ageSec}s]</span>{" "}
                      {entry.held && <span className="me-1 text-amber-300">held</span>}
                      {decodeInputLabel(truncated).map((label, labelIndex) => (
                        <span key={labelIndex} className={label.kind === "key" ? "me-1 rounded bg-wc-surface-input px-1" : ""}>
                          {label.kind === "text" ? `“${label.label}”` : label.label}
                        </span>
                      ))}
                      {discardPendingInput && <button type="button" className="ms-2 text-wc-text-muted hover:text-red-300" aria-label={`Discard pending input ${idx + 1}`} onClick={() => discardPendingInput(idx)}>×</button>}
                    </li>
                  );
                })}
              </ul>
              <div className="flex gap-2 text-[10px]">
                {discardAllPendingInput && <button type="button" className="rounded border border-wc-default px-1.5 py-0.5 hover:bg-wc-surface-input" aria-label="Discard all pending input" onClick={discardAllPendingInput}>Discard all</button>}
                {flushPendingInputNow && <button type="button" className="rounded border border-wc-default px-1.5 py-0.5 hover:bg-wc-surface-input" aria-label="Send pending input now" onClick={flushPendingInputNow}>Send now</button>}
              </div>
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
          {/* The surface colour lives on the wrapper, not the textarea: the
              interim mirror sits behind the textarea, so an opaque textarea
              would paint over the very text it is meant to reveal. */}
          <div className="relative flex min-w-0 rounded bg-wc-surface-input">
            {/* Transparent border of the same width as the textarea's so the
                mirror's content box lines up with the one it is drawing for. */}
            <InterimTranscriptOverlay
              draft={draft}
              interim={voicePartialTranscript ?? ""}
              textareaRef={textareaRef}
              className={cn(
                "box-border rounded border border-transparent px-2 text-base leading-5 text-wc-text-primary",
                onExpandComposer ? "pe-14" : "pe-2",
              )}
              testId="mobile-interim-overlay"
            />
            <Textarea
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
                "relative z-10 min-h-11 min-w-0 flex-1 resize-none rounded border border-wc-default bg-transparent px-2 text-base caret-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent",
                "overflow-y-auto overflow-x-hidden",
                // Hand the glyphs to the mirror while a hypothesis is on
                // screen so settled and unsettled text cannot double up.
                voicePartialTranscript ? "text-transparent" : "text-wc-text-primary",
                // Reserve room for the corner expand icon so long text never
                // slides under it.
                onExpandComposer && "pe-7",
              )}
              style={{
                minHeight: "44px",
                paddingBlock: "11px",
                paddingInlineStart: "8px",
                paddingInlineEnd: onExpandComposer ? "56px" : "8px",
                background: "transparent",
                borderColor: "rgb(var(--wc-border-default))",
                color: voicePartialTranscript ? "transparent" : "rgb(var(--wc-text-primary))",
                lineHeight: `${LINE_HEIGHT_PX}px`,
                maxHeight: `${MAX_TEXTAREA_HEIGHT}px`,
              }}
            />
            {/* Always-visible, low-emphasis corner icon that expands the draft
                into the full-screen composer. Keyboard-reachable and shown in
                both terminal and messages views — the composer is view-agnostic
                and the long-message pain applies equally when chatting. */}
            {onExpandComposer && (
              <IconButton
                type="button"
                data-testid="expand-toggle"
                aria-label={t(strings.mobileToolbar.expandComposerTitle)}
                onPointerDown={(e) => e.preventDefault()}
                onClick={onExpandComposer}
                shape="rounded"
                // `lg` renders a 24px glyph; this affordance has always been a
                // 16px icon sitting inside the composer's trailing gutter.
                size="sm"
                className="absolute inset-y-0 end-0 z-20"
                title={t(strings.mobileToolbar.expandComposerTitle)}
              >
                <Maximize2 aria-hidden className="h-4 w-4" />
              </IconButton>
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
        <ToolbarIconButton
          testId="mobile-command-submit"
          label={sendStatus === "sending" ? t(strings.mobileToolbar.statusSending) : t(strings.mobileToolbar.sendTitle)}
          onClick={submitCommand}
          className="shrink-0"
        >
          {/* "sending" renders as an inline spinner in the button itself rather
              than a label below the textarea — the label changed the toolbar's
              height on every send, forcing a costly terminal resize/reflow. */}
          {sendStatus === "sending" ? (
            <Loader2 data-testid="send-status-sending" aria-hidden className="h-4 w-4 animate-spin" />
          ) : (
            <SendHorizontal aria-hidden className="h-4 w-4" />
          )}
        </ToolbarIconButton>
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
      <div ref={keysAreaRef} className="min-w-0">
        <ToolbarSurface
          testId={viewMode === "messages" ? "messages-toolbar-actions" : "toolbar-keys-area"}
          layout={layout}
          ctx={controlContext}
          onMouseDown={(e) => e.preventDefault()}
        />
      </div>
    </div>
  );
});
