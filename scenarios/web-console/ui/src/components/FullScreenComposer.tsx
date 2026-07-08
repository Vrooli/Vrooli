import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { ImagePlus, Loader2, SendHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DrawerShell } from "./DrawerShell";
import { AttachmentPreviewTray, type ComposerAttachment } from "./composer/AttachmentPreviewTray";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { composeComposerPayload } from "../lib/composerPayload";
import type { ComposerDraft } from "../hooks/useComposerDraft";
import type { GateResult, InputSource } from "./terminal/inputGate";

type ComposerSendStatus = "idle" | "uploading" | "sending" | "queued" | "failed";

interface FullScreenComposerProps {
  /** Whether the composer overlay is open. */
  open: boolean;
  /** Minimize the composer WITHOUT losing the draft (Escape/backdrop/close/send). */
  onClose: () => void;
  /** Shared per-session draft (single source of truth for the text). */
  draft: ComposerDraft;
  /** Inject the composed payload into the active terminal via the input gate. */
  onInput: (data: string, source: InputSource) => GateResult;
  /** Subscribe to per-send settlement so we only clear+minimize on ok. */
  subscribeInputSettled?: (cb: (seq: number, ok: boolean) => void) => () => void;
  /** Move focus to the active terminal (e.g. after a successful send). */
  onFocusTerminal?: () => void;

  /** Mic control (VoiceMicButton) rendered by the parent so voice stays unmodified. */
  mic?: ReactNode;

  /** Staged image attachments. */
  attachments?: ComposerAttachment[];
  /** Stage newly picked image files (thumbnails only — never uploaded here). */
  onAttachFiles?: (files: File[]) => void;
  /** Remove a single staged attachment. */
  onRemoveAttachment?: (id: string) => void;
  /**
   * Upload every staged file and resolve their terminal paths, in attachment
   * order. Called on send; rejects to keep the attachments + text intact.
   */
  resolveAttachmentPaths?: () => Promise<string[]>;
  /** Clear the staged attachments after a successful send. */
  onClearAttachments?: () => void;
}

/**
 * FullScreenComposer — a portaled DrawerShell overlay for authoring long,
 * mixed text+image messages. It is an OVERLAY, not a pane replacement: the
 * xterm terminal stays mounted underneath and never reflows.
 *
 * The textarea is uncontrolled and bound to the shared `draft`, so the
 * collapsed toolbar input and this composer read/write one value that cannot
 * diverge. Terminal keys/modifiers are intentionally absent — this is a message
 * composer, not a terminal-key surface.
 */
export default function FullScreenComposer({
  open,
  onClose,
  draft,
  onInput,
  subscribeInputSettled,
  onFocusTerminal,
  mic,
  attachments = [],
  onAttachFiles,
  onRemoveAttachment,
  resolveAttachmentPaths,
  onClearAttachments,
}: FullScreenComposerProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<ComposerSendStatus>("idle");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [showDiscardPrompt, setShowDiscardPrompt] = useState(false);
  const settlementUnsubRef = useRef<(() => void) | null>(null);

  // Bind the uncontrolled textarea to the shared draft: reseed on peer changes
  // (voice, session reload, collapsed-input edits) without clobbering our caret.
  useEffect(() => {
    if (!open) return;
    return draft.subscribe((change) => {
      const el = textareaRef.current;
      const isOwnTyping = change.reason === "input" && el != null && document.activeElement === el;
      if (el && !isOwnTyping && el.value !== change.value) {
        el.value = change.value;
        if (change.caret != null) {
          try {
            el.setSelectionRange(change.caret, change.caret);
          } catch {
            /* detached during teardown */
          }
        }
      }
    });
  }, [draft, open]);

  // Auto-focus the textarea on open and restore focus to the opener on close.
  const openerRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!open) return;
    openerRef.current = (document.activeElement as HTMLElement) ?? null;
    setStatus("idle");
    setErrorMsg(null);
    setShowDiscardPrompt(false);
    const raf = requestAnimationFrame(() => {
      const el = textareaRef.current;
      if (el) {
        el.value = draft.getValue();
        el.focus();
        const end = el.value.length;
        try {
          el.setSelectionRange(end, end);
        } catch {
          /* ignore */
        }
      }
    });
    return () => {
      cancelAnimationFrame(raf);
      const opener = openerRef.current;
      if (opener && typeof opener.focus === "function") opener.focus();
    };
  }, [open, draft]);

  // Cancel any dangling settlement subscription on unmount.
  useEffect(() => {
    return () => {
      settlementUnsubRef.current?.();
      settlementUnsubRef.current = null;
    };
  }, []);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      draft.handleChange(e.currentTarget);
    },
    [draft],
  );

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

  const hasAttachments = attachments.length > 0;

  // Minimizing with staged (never-sent) images would silently strand them, so
  // intercept close and prompt to discard first. Send/clear paths call onClose
  // directly (after clearing), so they never hit this prompt.
  const requestClose = useCallback(() => {
    if (hasAttachments) {
      setShowDiscardPrompt(true);
      return;
    }
    onClose();
  }, [hasAttachments, onClose]);

  const confirmDiscard = useCallback(() => {
    onClearAttachments?.();
    setShowDiscardPrompt(false);
    onClose();
  }, [onClearAttachments, onClose]);

  const handleFilesPicked = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(e.target.files ?? []);
      e.target.value = "";
      if (files.length > 0) onAttachFiles?.(files);
    },
    [onAttachFiles],
  );

  const handleSend = useCallback(async () => {
    if (status === "sending" || status === "uploading") return;
    const text = draft.getValue();
    if (text.length === 0 && !hasAttachments) return;
    setErrorMsg(null);

    // Upload staged files first so a failure never clears text/attachments.
    let paths: string[] = [];
    if (hasAttachments && resolveAttachmentPaths) {
      setStatus("uploading");
      try {
        paths = await resolveAttachmentPaths();
      } catch {
        setStatus("failed");
        setErrorMsg(t(strings.composer.uploadFailed));
        return;
      }
    }

    const payload = composeComposerPayload(text, paths);
    const result = onInput(payload, "toolbar-submit");

    if (result.status === "rejected") {
      // "disposed" — pane torn down. Keep the draft/attachments; go idle.
      setStatus("idle");
      return;
    }
    if (result.status === "queued") {
      // Not sent immediately (connection lost / gate paused). Per the send
      // contract we only clear+minimize on an ok settlement, so keep the draft
      // and attachments and surface the queued state.
      setStatus("queued");
      return;
    }

    setStatus("sending");
    const finalizeSuccess = () => {
      draft.reset();
      onClearAttachments?.();
      setStatus("idle");
      onClose();
      onFocusTerminal?.();
    };
    const finalizeFailure = () => {
      setStatus("failed");
      setErrorMsg(t(strings.composer.sendFailed));
    };

    if (subscribeInputSettled) {
      awaitNextSettlement((ok) => (ok ? finalizeSuccess() : finalizeFailure()));
    } else {
      finalizeSuccess();
    }
  }, [
    status,
    draft,
    hasAttachments,
    resolveAttachmentPaths,
    onInput,
    subscribeInputSettled,
    awaitNextSettlement,
    onClearAttachments,
    onClose,
    onFocusTerminal,
    t,
  ]);

  const isBusy = status === "sending" || status === "uploading";
  const canSend = true; // empty+no-attachments is guarded inside handleSend

  return (
    <DrawerShell
      open={open}
      onClose={requestClose}
      closeAriaLabel={t(strings.composer.closeAriaLabel)}
      title={t(strings.composer.title)}
      panelTestId="full-screen-composer"
      avoidKeyboard
    >
      <div className="relative flex h-full flex-col">
        <textarea
          ref={textareaRef}
          data-testid="composer-input"
          defaultValue={draft.getValue()}
          onChange={handleChange}
          onSelect={(e) => draft.trackSelection(e.currentTarget)}
          onBlur={(e) => draft.trackSelection(e.currentTarget)}
          autoComplete="off"
          autoCorrect="on"
          spellCheck
          placeholder={t(strings.composer.placeholder)}
          className="min-h-0 flex-1 resize-none bg-transparent px-4 py-3 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none"
        />

        {(status === "queued" || status === "failed") && errorMsg !== null && (
          <div
            data-testid="composer-error"
            className={cn(
              "px-4 py-1 text-xs",
              status === "failed" ? "text-red-400" : "text-yellow-400",
            )}
          >
            {errorMsg}
          </div>
        )}
        {status === "queued" && errorMsg === null && (
          <div data-testid="composer-status-queued" className="px-4 py-1 text-xs text-yellow-400">
            {t(strings.mobileToolbar.statusQueued)}
          </div>
        )}

        {onRemoveAttachment && (
          <div className="px-4">
            <AttachmentPreviewTray
              attachments={attachments}
              onRemove={onRemoveAttachment}
              removeAriaLabel={t(strings.composer.removeAttachmentAriaLabel)}
            />
          </div>
        )}

        {/* items-stretch so the attach + mic buttons take the send button's
            height automatically (the send button is the tallest child) — no
            hard-coded heights to drift out of sync. */}
        <div className="flex items-stretch gap-2 border-t border-wc-default px-3 pt-2 pb-[max(0.5rem,var(--wc-safe-bottom,0px))]">
          {onAttachFiles && (
            <>
              <button
                type="button"
                data-testid="composer-attach"
                onClick={() => fileInputRef.current?.click()}
                disabled={isBusy}
                className="flex shrink-0 items-center justify-center rounded border border-wc-default bg-wc-surface-input p-2 text-wc-text-secondary transition hover:text-wc-text-primary disabled:opacity-50"
                title={t(strings.composer.attachImageTitle)}
                aria-label={t(strings.composer.attachImageTitle)}
              >
                <ImagePlus className="h-4 w-4" />
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                multiple
                hidden
                data-testid="composer-file-input"
                onChange={handleFilesPicked}
              />
            </>
          )}
          {mic}
          <div className="min-w-0 flex-1" />
          <button
            type="button"
            data-testid="composer-send"
            onClick={handleSend}
            disabled={!canSend || isBusy}
            className="inline-flex shrink-0 items-center gap-1.5 rounded border border-wc-accent bg-wc-accent/20 px-4 py-2 text-sm font-medium text-wc-text-primary transition active:bg-wc-accent-active disabled:opacity-60"
            title={isBusy ? t(strings.composer.sendingTitle) : t(strings.composer.sendTitle)}
          >
            {isBusy ? (
              <Loader2 data-testid="composer-sending" className="h-4 w-4 animate-spin" />
            ) : (
              <SendHorizontal className="h-4 w-4" />
            )}
            <span>{t(strings.composer.sendTitle)}</span>
          </button>
        </div>

        {showDiscardPrompt && (
          <div
            data-testid="composer-discard-prompt"
            className="absolute inset-0 z-[90] flex items-center justify-center bg-wc-backdrop p-4"
            onClick={() => setShowDiscardPrompt(false)}
          >
            <div
              className="wc-stable-theme w-full max-w-sm rounded-lg border border-wc-default bg-wc-surface-raised p-5 shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <h2 className="mb-2 text-sm font-semibold text-wc-text-primary">
                {t(strings.composer.discardTitle)}
              </h2>
              <p className="mb-4 text-xs text-wc-text-secondary">
                {t(strings.composer.discardMessage, { count: attachments.length })}
              </p>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  data-testid="composer-discard-cancel"
                  onClick={() => setShowDiscardPrompt(false)}
                  className="rounded-full px-4 py-1.5 text-sm font-medium text-wc-text-primary transition hover:bg-white/10"
                >
                  {t(strings.composer.discardCancel)}
                </button>
                <button
                  type="button"
                  data-testid="composer-discard-confirm"
                  onClick={confirmDiscard}
                  className="rounded-full bg-red-600 px-4 py-1.5 text-sm font-medium text-white transition hover:bg-red-700"
                >
                  {t(strings.composer.discardConfirm)}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DrawerShell>
  );
}
