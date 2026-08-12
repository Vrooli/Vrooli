import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useDeferredValue, useRef, useState, useEffect, forwardRef, useImperativeHandle } from "react";
import { Image, Loader2, Maximize2, SendHorizontal, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { TOOLBAR_KEYS, ESC_KEY, TAB_KEY, ENTER_KEY, ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, applyModifiers } from "../consts/toolbar-keys";
import { strings } from "../consts/strings";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import KeyComboPicker from "./KeyComboPicker";
import VoiceMicButton from "./VoiceMicButton";
import VoiceCommandSuggestion from "./VoiceCommandSuggestion";
import AiSuggestBar from "./AiSuggestBar";
import { slugify } from "../lib/slugify";
import { useComposerDraft } from "../hooks/useComposerDraft";
import { useHoldRepeat } from "../hooks/useHoldRepeat";
/**
 * Arrow keys are the only toolbar buttons with hold-to-repeat because users
 * routinely need to scan through shell history, long command lines, and TUI
 * views. Other toolbar buttons (Esc/Tab/Enter, modifiers) are one-shot
 * actions where repeat would only cause accidental misfires.
 */
const ARROW_KEYS = new Set([ARROW_UP.input, ARROW_DOWN.input, ARROW_LEFT.input, ARROW_RIGHT.input]);
/**
 * A toolbar arrow button that fires on pointerdown and auto-repeats while
 * held (via useHoldRepeat). Intentionally does NOT bind onClick — pointerdown
 * already dispatches, and a parallel click handler would double-fire on tap.
 */
function ArrowToolbarButton({ keyDef, onFire, className }) {
    const handlers = useHoldRepeat({ onFire: useCallback(() => onFire(keyDef), [onFire, keyDef]) });
    return (_jsx("button", { "data-testid": `toolbar-key-${slugify(keyDef.label)}`, tabIndex: -1, ...handlers, className: className, children: keyDef.label }));
}
/**
 * Wraps AiSuggestBar and applies useDeferredValue to the draft text *here*
 * rather than in MobileToolbar. Hoisting the deferral into a child that only
 * mounts while the suggest bar is open means a normal keystroke no longer
 * triggers MobileToolbar's deferred (second) render pass when the bar is
 * closed — which is the common case and was adding latency to typing.
 */
function DeferredAiSuggestBar({ inputText, onExecute, onClose, }) {
    const deferredInputText = useDeferredValue(inputText);
    return _jsx(AiSuggestBar, { inputText: deferredInputText, onExecute: onExecute, onClose: onClose });
}
// [REQ:P0-007a] Floating Toolbar Component
// [REQ:P0-007b] Terminal Key/Chord Mapping
/** Max visible lines before the textarea stops growing. */
const MAX_VISIBLE_LINES = 4;
/** Approximate line height in px for the textarea. */
const LINE_HEIGHT_PX = 20;
/** Max textarea height: MAX_VISIBLE_LINES * line-height + padding. */
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;
export default forwardRef(function MobileToolbar({ onInput, subscribeInputSettled, subscribePendingInput, getPendingInputSnapshot, onFocusTerminal, activeSessionId, draft: draftProp, onExpandComposer, visible = true, voiceSupported, voicePreparing, voiceRecording, voicePersistentMode, voiceListening, voicePassive, voiceTranscribing, voiceStaleLiveMic, voiceError, voiceLevel, voiceActivity, voicePartialTranscript, voiceBackend, voiceCanExportDiagnostic, onVoiceExportDiagnostic, onVoiceStart, onVoicePrepare, onVoiceStop, onVoiceExitPassive, onVoiceReleaseMic, onVoiceCancel, voiceCommandSuggestion, onVoiceCommandConfirm, onVoiceCommandDismiss, onUploadImage, onOpenAi, aiSuggestActive, onAiSuggestExecute, isTtsSpeaking, onTtsStop, viewMode = "terminal", onSwitchToTerminal, }, ref) {
    const { t } = useTranslation();
    // Single-source the draft. When Workspace passes a shared draft, the collapsed
    // input and the full-screen composer read/write the same value; the private
    // fallback keeps the toolbar usable standalone (and in unit tests). Both hooks
    // run unconditionally (hooks rule) but only the selected one is wired up.
    const fallbackDraft = useComposerDraft(activeSessionId);
    const draft = draftProp ?? fallbackDraft;
    const textareaRef = useRef(null);
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
        if (aiSuggestActive)
            setAiInputText(draft.getValue());
    }, [aiSuggestActive, draft]);
    // Auto-resize the textarea to fit its content (up to MAX_TEXTAREA_HEIGHT).
    // Reset to "auto" first so scrollHeight reflects the natural content height,
    // then only write back when it actually changed to avoid a redundant layout.
    const resizeTextarea = useCallback(() => {
        const el = textareaRef.current;
        if (!el)
            return;
        el.style.height = "auto";
        const target = Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT);
        const next = `${target}px`;
        if (el.style.height !== next)
            el.style.height = next;
    }, []);
    // Track the live value + caret as the user types. No setState in the common
    // path → no re-render per keystroke. The shared draft notifies peer surfaces.
    const handleTextareaChange = useCallback((e) => {
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
                if (el.value !== change.value)
                    el.value = change.value;
                if (change.caret != null) {
                    try {
                        el.setSelectionRange(change.caret, change.caret);
                    }
                    catch {
                        // The textarea may be detached during teardown; ignore.
                    }
                }
                resizeTextarea();
            }
            if (aiSuggestActiveRef.current)
                setAiInputText(change.value);
        });
    }, [draft, resizeTextarea]);
    useImperativeHandle(ref, () => ({
        appendText: (text) => {
            draft.appendAtCaret(text);
        },
        focusInput: () => {
            textareaRef.current?.focus();
        },
        clearInput: () => {
            draft.reset();
        },
    }), [draft]);
    const [sendStatus, setSendStatus] = useState("idle");
    /** Draft snapshot taken at submit time; restored on ack failure. */
    const pendingSendRef = useRef(null);
    /** Unsubscribe for the current in-flight settlement subscription. */
    const settlementUnsubRef = useRef(null);
    const [pendingInputEntries, setPendingInputEntries] = useState([]);
    const [pillOpen, setPillOpen] = useState(false);
    const toolbarLayout = useWorkspaceStore((s) => s.toolbarLayout);
    const modifiers = useWorkspaceStore((s) => s.modifiers);
    const toggleModifier = useWorkspaceStore((s) => s.toggleModifier);
    const clearModifiers = useWorkspaceStore((s) => s.clearModifiers);
    const statusTimerRef = useRef(null);
    // Clear status timer on unmount
    useEffect(() => {
        return () => {
            if (statusTimerRef.current !== null)
                clearTimeout(statusTimerRef.current);
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
    const showStatus = useCallback((status) => {
        setSendStatus(status);
        if (statusTimerRef.current !== null)
            clearTimeout(statusTimerRef.current);
        if (status !== "idle") {
            statusTimerRef.current = setTimeout(() => {
                statusTimerRef.current = null;
                setSendStatus("idle");
            }, 2500);
        }
    }, []);
    const handleKey = useCallback((key) => {
        const mods = useWorkspaceStore.getState().modifiers;
        const { data, consumed } = applyModifiers(key.input, mods);
        onInput(data, "toolbar-key");
        if (consumed)
            clearModifiers();
        // Re-focus the terminal after sending the key. On mobile, rapid taps can
        // cause the browser to blur the terminal (moving activeElement to body),
        // which dismisses the virtual keyboard. Since the user is actively pressing
        // toolbar keys, they clearly want to interact with the terminal, so
        // restoring focus is always correct here. This does NOT focus the
        // MobileToolbar's textarea — it focuses the xterm.js terminal.
        onFocusTerminal?.();
    }, [onInput, clearModifiers, onFocusTerminal]);
    /**
     * Subscribe to the next single settlement event from the terminal socket.
     * The subscription auto-unsubscribes after it fires once — subsequent
     * acks from other senders (e.g. xterm direct keystrokes) are ignored.
     */
    const awaitNextSettlement = useCallback((onSettle) => {
        if (!subscribeInputSettled)
            return;
        settlementUnsubRef.current?.();
        const unsub = subscribeInputSettled((_seq, ok) => {
            settlementUnsubRef.current?.();
            settlementUnsubRef.current = null;
            onSettle(ok);
        });
        settlementUnsubRef.current = unsub;
    }, [subscribeInputSettled]);
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
            if (viewMode === "messages")
                onSwitchToTerminal?.();
            onFocusTerminal?.();
            return;
        }
        const mods = useWorkspaceStore.getState().modifiers;
        const hasModifier = mods.ctrl || mods.alt || mods.shift;
        let dataToSend;
        if (hasModifier) {
            // With modifiers active, apply them to the input text character by character
            // (useful for combos like Ctrl+C, Ctrl+Alt+2, etc.)
            const { data } = applyModifiers(draftText, mods);
            dataToSend = data;
            clearModifiers();
        }
        else {
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
            if (viewMode === "messages")
                onSwitchToTerminal?.();
        };
        const finalizeFailure = () => {
            pendingSendRef.current = null;
            // Draft is still in the textarea (we never cleared it); just surface
            // the failure so the user can retry by pressing Send again.
            showStatus("failed");
        };
        if (subscribeInputSettled) {
            awaitNextSettlement((ok) => {
                if (ok)
                    finalizeSuccess();
                else
                    finalizeFailure();
            });
        }
        else {
            // No settlement seam available (legacy caller wiring) — fall back
            // to the optimistic behavior so we don't strand the draft.
            finalizeSuccess();
        }
        // After submitting a command, focus the terminal so the user can
        // immediately see and interact with the output.
        onFocusTerminal?.();
    }, [onInput, draft, showStatus, onFocusTerminal, clearModifiers, viewMode, onSwitchToTerminal, subscribeInputSettled, awaitNextSettlement]);
    if (!visible)
        return null;
    return (_jsxs("div", { "data-testid": "mobile-toolbar", 
        // Do not add bottom safe-area padding here. The toolbar is part of the
        // fixed app layout, and iOS/PWA safe-area handling already reserves the
        // bottom edge; adding it here creates a visible extra gutter.
        className: "wc-chrome-surface-raised flex shrink-0 flex-col border-t border-wc-default touch-manipulation ps-[max(0.25rem,var(--wc-safe-left,0px))] pe-[max(0.25rem,var(--wc-safe-right,0px))]", children: [pendingInputEntries.length > 0 && (_jsxs("div", { "data-testid": "pending-input-pill", className: "flex flex-col border-b border-wc-default bg-wc-surface-raised/80 px-2 py-1 text-[11px] text-yellow-300", children: [_jsxs("button", { type: "button", onPointerDown: (e) => e.preventDefault(), onClick: () => setPillOpen((v) => !v), className: "flex items-center justify-between gap-2 text-start", title: t(strings.mobileToolbar.showUnsentTitle), children: [_jsxs("span", { children: ["\u23F3 ", t(strings.mobileToolbar.unsentCount, { count: pendingInputEntries.length }), (() => {
                                        const oldest = pendingInputEntries.reduce((min, e) => (min === null || e.addedAt < min ? e.addedAt : min), null);
                                        if (oldest === null)
                                            return null;
                                        const ageSec = Math.max(0, Math.floor((Date.now() - oldest) / 1000));
                                        return _jsx("span", { className: "ms-1 text-wc-text-muted", children: t(strings.mobileToolbar.unsentOldest, { seconds: ageSec }) });
                                    })()] }), _jsx("span", { className: "text-wc-text-muted", children: pillOpen ? "▾" : "▸" })] }), pillOpen && (_jsx("div", { "data-testid": "pending-input-disclosure", className: "mt-1 flex flex-col gap-1", children: _jsx("ul", { className: "max-h-24 overflow-y-auto font-mono text-[10px] text-wc-text-secondary", children: pendingInputEntries.map((entry, idx) => {
                                const truncated = entry.data.length > 60 ? entry.data.slice(0, 60) + "…" : entry.data;
                                const ageSec = Math.max(0, Math.floor((Date.now() - entry.addedAt) / 1000));
                                return (_jsxs("li", { className: "truncate", children: [_jsxs("span", { className: "text-wc-text-muted", children: ["[", ageSec, "s]"] }), " ", truncated.replace(/\n/g, "⏎")] }, idx));
                            }) }) }))] })), voiceCommandSuggestion && onVoiceCommandConfirm && onVoiceCommandDismiss && (_jsx(VoiceCommandSuggestion, { suggestion: voiceCommandSuggestion, onConfirm: onVoiceCommandConfirm, onDismiss: onVoiceCommandDismiss })), aiSuggestActive && onAiSuggestExecute && (_jsx(DeferredAiSuggestBar, { inputText: aiInputText, onExecute: onAiSuggestExecute, onClose: () => onOpenAi?.() })), _jsxs("div", { className: "flex items-end gap-0.5 px-1 py-1", children: [_jsxs("div", { className: "flex min-w-0 flex-1 flex-col", children: [_jsxs("div", { className: "relative flex min-w-0", children: [_jsx("textarea", { ref: textareaRef, "data-testid": "mobile-command-input", defaultValue: draft.getValue(), onChange: handleTextareaChange, onSelect: (e) => draft.trackSelection(e.currentTarget), onBlur: (e) => draft.trackSelection(e.currentTarget), autoComplete: "off", autoCorrect: "on", spellCheck: false, rows: 1, placeholder: t(strings.mobileToolbar.placeholder), className: cn("min-w-0 flex-1 resize-none rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent", "overflow-y-auto overflow-x-hidden", 
                                        // Reserve room for the corner expand icon so long text never
                                        // slides under it.
                                        onExpandComposer && "pe-7"), style: {
                                            lineHeight: `${LINE_HEIGHT_PX}px`,
                                            maxHeight: `${MAX_TEXTAREA_HEIGHT}px`,
                                        } }), onExpandComposer && (_jsx("button", { type: "button", "data-testid": "expand-toggle", onPointerDown: (e) => e.preventDefault(), onClick: onExpandComposer, className: "absolute end-1 top-1 rounded p-0.5 text-wc-text-muted/70 transition hover:bg-wc-surface-raised hover:text-wc-text-primary", title: t(strings.mobileToolbar.expandComposerTitle), "aria-label": t(strings.mobileToolbar.expandComposerTitle), children: _jsx(Maximize2, { className: "h-3.5 w-3.5" }) }))] }), sendStatus === "queued" && (_jsx("span", { "data-testid": "send-status-queued", className: "px-1 text-[10px] text-yellow-400", children: t(strings.mobileToolbar.statusQueued) })), sendStatus === "failed" && (_jsx("span", { "data-testid": "send-status-failed", className: "px-1 text-[10px] text-red-400", children: t(strings.mobileToolbar.statusFailed) }))] }), _jsx("button", { "data-testid": "mobile-command-submit", onPointerDown: (e) => e.preventDefault(), onClick: submitCommand, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: sendStatus === "sending" ? t(strings.mobileToolbar.statusSending) : t(strings.mobileToolbar.sendTitle), children: sendStatus === "sending" ? (_jsx(Loader2, { "data-testid": "send-status-sending", className: "h-3.5 w-3.5 animate-spin" })) : (_jsx(SendHorizontal, { className: "h-3.5 w-3.5" })) })] }), viewMode === "messages" ? (
            /* ── Messages mode: AI + image upload + voice mic ── */
            _jsxs("div", { "data-testid": "messages-toolbar-actions", className: "flex items-stretch gap-0.5 px-1 py-1 touch-manipulation select-none", onMouseDown: (e) => e.preventDefault(), children: [onOpenAi && (_jsx("button", { "data-testid": "toolbar-ai", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onOpenAi, className: cn("flex min-w-0 flex-1 items-center justify-center rounded border p-1.5 transition active:bg-wc-accent-active touch-manipulation", aiSuggestActive
                            ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                            : "border-wc-default bg-wc-surface-input text-wc-text-secondary"), title: t(strings.mobileToolbar.aiCommandTitle), children: _jsx(Sparkles, { className: "h-3.5 w-3.5" }) })), onUploadImage && (_jsx("button", { "data-testid": "toolbar-upload-image", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onUploadImage, className: "flex min-w-0 flex-1 items-center justify-center rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: t(strings.mobileToolbar.uploadImageTitle), children: _jsx(Image, { className: "h-3.5 w-3.5" }) })), voiceSupported && onVoiceStart && onVoiceStop && (_jsx(VoiceMicButton, { testId: "voice-mic-btn", supported: voiceSupported, isPreparing: voicePreparing ?? false, isRecording: voiceRecording ?? false, persistentMode: voicePersistentMode ?? false, isListening: voiceListening ?? false, size: "sm", isPassive: voicePassive ?? false, isTranscribing: voiceTranscribing ?? false, staleLiveMic: voiceStaleLiveMic ?? false, error: voiceError ?? null, audioLevel: voiceLevel, voiceActivity: voiceActivity, partialTranscript: voicePartialTranscript, backend: voiceBackend, canExportDiagnostic: voiceCanExportDiagnostic, onExportDiagnostic: onVoiceExportDiagnostic, isTtsSpeaking: isTtsSpeaking, onPrepare: onVoicePrepare, onStart: onVoiceStart, onStop: onVoiceStop, onExitPassive: onVoiceExitPassive, onReleaseMic: onVoiceReleaseMic, onCancel: onVoiceCancel, onTtsStop: onTtsStop, className: "flex min-w-0 flex-1", buttonClassName: "flex w-full items-center justify-center" }))] })) : toolbarLayout === "expanded" ? (
            /* ── Expanded layout: two rows with D-pad arrow cluster ──
               ┌────────────────────────────────────────────────────────────┐
               │ [Ctrl] [Alt] [Shift] │     [↑]      │ [📷] │            │
               │ [Esc]  [Tab] [Enter] │ [←] [↓] [→]  │      │    [🎤]   │
               └────────────────────────────────────────────────────────────┘
               The mic button spans both rows for easy access. */
            _jsxs("div", { className: "grid gap-0.5 px-1 py-1 touch-manipulation select-none", 
                // Mic column is `minmax(0,1fr)` so it grows to fill remaining width
                // (up to ~2× the image-upload button) but is allowed to shrink when
                // the other columns are wide — preventing viewport overflow.
                style: { gridTemplateColumns: "auto auto auto minmax(0,1fr)", gridTemplateRows: "auto auto" }, onMouseDown: (e) => e.preventDefault(), children: [_jsxs("div", { className: "flex flex-col gap-0.5", style: { gridRow: "1 / -1" }, children: [_jsxs("div", { className: "flex items-stretch gap-0.5", children: [_jsx(KeyComboPicker, { onInput: onInput, onFocusTerminal: onFocusTerminal, triggerClassName: "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation flex items-center justify-center" }), _jsx("div", { className: "w-px self-stretch bg-wc-default shrink-0" }), ["ctrl", "alt", "shift"].map((mod) => (_jsx("button", { "data-testid": `toolbar-mod-${mod}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => toggleModifier(mod), className: cn("shrink-0 rounded border px-2 py-1.5 text-sm font-medium transition touch-manipulation", modifiers[mod]
                                            ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                                            : "border-wc-default bg-wc-surface-input text-wc-text-secondary active:bg-wc-accent-active"), children: mod.charAt(0).toUpperCase() + mod.slice(1) }, mod)))] }), _jsx("div", { className: "flex items-center gap-0.5", children: [ESC_KEY, TAB_KEY, ENTER_KEY].map((key) => (_jsx("button", { "data-testid": `toolbar-key-${slugify(key.label)}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => handleKey(key), className: "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.75rem]", children: key.label }, key.label))) })] }), _jsxs("div", { className: "flex flex-col items-center gap-0.5 px-1", style: { gridRow: "1 / -1" }, children: [_jsx("div", { className: "flex justify-center", children: _jsx(ArrowToolbarButton, { keyDef: ARROW_UP, onFire: handleKey, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]" }) }), _jsx("div", { className: "flex items-center gap-0.5", children: [ARROW_LEFT, ARROW_DOWN, ARROW_RIGHT].map((key) => (_jsx(ArrowToolbarButton, { keyDef: key, onFire: handleKey, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-sm font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation min-w-[2.25rem]" }, key.label))) })] }), _jsxs("div", { className: "flex flex-col items-end gap-0.5", style: { gridColumn: 3, gridRow: "1 / -1" }, children: [onOpenAi && (_jsx("button", { "data-testid": "toolbar-ai", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onOpenAi, className: cn("shrink-0 rounded border p-2 transition active:bg-wc-accent-active touch-manipulation", aiSuggestActive
                                    ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                                    : "border-wc-default bg-wc-surface-input text-wc-text-secondary"), title: t(strings.mobileToolbar.aiCommandTitle), children: _jsx(Sparkles, { className: "h-4 w-4" }) })), onUploadImage && (_jsx("button", { "data-testid": "toolbar-upload-image", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onUploadImage, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-2 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: t(strings.mobileToolbar.uploadImageTitle), children: _jsx(Image, { className: "h-4 w-4" }) }))] }), voiceSupported && onVoiceStart && onVoiceStop && (_jsx("div", { className: "flex items-stretch", style: { gridColumn: 4, gridRow: "1 / -1" }, children: _jsx(VoiceMicButton, { testId: "voice-mic-btn", supported: voiceSupported, isPreparing: voicePreparing ?? false, isRecording: voiceRecording ?? false, persistentMode: voicePersistentMode ?? false, isListening: voiceListening ?? false, size: "lg", isTranscribing: voiceTranscribing ?? false, staleLiveMic: voiceStaleLiveMic ?? false, error: voiceError ?? null, audioLevel: voiceLevel, voiceActivity: voiceActivity, partialTranscript: voicePartialTranscript, backend: voiceBackend, canExportDiagnostic: voiceCanExportDiagnostic, onExportDiagnostic: onVoiceExportDiagnostic, isTtsSpeaking: isTtsSpeaking, onPrepare: onVoicePrepare, onStart: onVoiceStart, onStop: onVoiceStop, onExitPassive: onVoiceExitPassive, onReleaseMic: onVoiceReleaseMic, onCancel: onVoiceCancel, onTtsStop: onTtsStop, className: "h-full w-full", 
                            // Mic stretches to fill the remaining row width (its grid
                            // column is minmax(0,1fr)), making it as wide as fits without
                            // pushing other buttons off-screen. min-width keeps it a
                            // usable tap target even on very narrow viewports.
                            buttonClassName: "h-full w-full min-w-[2.5rem] flex items-center justify-center" }) }))] })) : (
            /* ── Compact layout: single row (original) ── */
            _jsxs("div", { className: "flex items-center gap-0.5 px-1 py-1 touch-manipulation select-none", onMouseDown: (e) => e.preventDefault(), children: [_jsx(KeyComboPicker, { onInput: onInput, onFocusTerminal: onFocusTerminal }), _jsx("div", { className: "w-px h-4 bg-wc-default shrink-0" }), ["ctrl", "alt", "shift"].map((mod) => (_jsx("button", { "data-testid": `toolbar-mod-${mod}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => toggleModifier(mod), className: cn("shrink-0 rounded border px-1.5 py-1 text-xs font-medium transition touch-manipulation", modifiers[mod]
                            ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                            : "border-wc-default bg-wc-surface-input text-wc-text-secondary active:bg-wc-accent-active"), children: mod.charAt(0).toUpperCase() + mod.slice(1) }, mod))), _jsx("div", { className: "w-px h-4 bg-wc-default shrink-0" }), _jsx("div", { className: "flex items-center gap-0.5 overflow-x-auto min-w-0 flex-1", children: TOOLBAR_KEYS.map((key) => {
                            const className = cn("shrink-0 rounded border border-wc-default bg-wc-surface-input px-1.5 py-1 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", key.width === "wide" ? "min-w-[3.5rem]" : key.width === "narrow" ? "min-w-[1.75rem]" : "min-w-[2.25rem]");
                            if (ARROW_KEYS.has(key.input)) {
                                return (_jsx(ArrowToolbarButton, { keyDef: key, onFire: handleKey, className: className }, key.label));
                            }
                            return (_jsx("button", { "data-testid": `toolbar-key-${slugify(key.label)}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => handleKey(key), className: className, children: key.label }, key.label));
                        }) }), onOpenAi && (_jsx("button", { "data-testid": "toolbar-ai", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onOpenAi, className: cn("shrink-0 rounded border p-1.5 transition active:bg-wc-accent-active touch-manipulation", aiSuggestActive
                            ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary"
                            : "border-wc-default bg-wc-surface-input text-wc-text-secondary"), title: t(strings.mobileToolbar.aiCommandTitle), children: _jsx(Sparkles, { className: "h-3.5 w-3.5" }) })), onUploadImage && (_jsx("button", { "data-testid": "toolbar-upload-image", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: onUploadImage, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: t(strings.mobileToolbar.uploadImageTitle), children: _jsx(Image, { className: "h-3.5 w-3.5" }) })), voiceSupported && onVoiceStart && onVoiceStop && (_jsx(VoiceMicButton, { testId: "voice-mic-btn", supported: voiceSupported, isPreparing: voicePreparing ?? false, isRecording: voiceRecording ?? false, persistentMode: voicePersistentMode ?? false, isListening: voiceListening ?? false, size: "sm", isPassive: voicePassive ?? false, isTranscribing: voiceTranscribing ?? false, staleLiveMic: voiceStaleLiveMic ?? false, error: voiceError ?? null, audioLevel: voiceLevel, voiceActivity: voiceActivity, partialTranscript: voicePartialTranscript, backend: voiceBackend, canExportDiagnostic: voiceCanExportDiagnostic, onExportDiagnostic: onVoiceExportDiagnostic, isTtsSpeaking: isTtsSpeaking, onPrepare: onVoicePrepare, onStart: onVoiceStart, onStop: onVoiceStop, onExitPassive: onVoiceExitPassive, onReleaseMic: onVoiceReleaseMic, onCancel: onVoiceCancel, onTtsStop: onTtsStop }))] }))] }));
});
