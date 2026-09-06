/**
 * @libraryId react-component-library:PromptComposer
 * @displayName PromptComposer
 * @description The message composer with auto-resizing entry, attachments, slash commands, mentions, model and tool context, submit, stop, retry, draft persistence, and keyboard behavior that never traps the user.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource ai.prompt-composer */
import {
  useId,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { CommandButton } from "@vrooli/react-component-library/CommandButton/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import {
  ComposerAttachmentTray,
  type ComposerAttachment,
} from "@vrooli/react-component-library/ComposerAttachmentTray/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import type { CommandRegistry } from "@vrooli/react-component-library/CommandRegistry/1";

export interface ComposerSubmission {
  text: string;
  attachmentIds: string[];
}
export interface PromptComposerProps {
  /** Keep this state per conversation in the caller to preserve drafts across navigation. */
  value: string;
  onValueChange: (value: string) => void;
  /** Return true only after the transport accepts this exact submission. */
  onSend: (submission: ComposerSubmission) => Promise<boolean>;
  onSent?: (submission: ComposerSubmission) => void;
  attachments?: ComposerAttachment[];
  onFiles?: (files: File[]) => void;
  onAttachmentRemove?: (id: string) => void;
  onAttachmentRetry?: (id: string) => void;
  onAttachmentCancel?: (id: string) => void;
  onAttachmentReorder?: (ids: string[]) => void;
  commands?: CommandRegistry;
  context?: ReactNode;
  label?: string;
  placeholder?: string;
  disabledReason?: string;
  offline?: boolean;
  generating?: boolean;
  onStop?: () => Promise<unknown>;
  maxLength?: number;
  accept?: string;
  sendLabel?: string;
  className?: string;
}
const css = `
[data-rcl-prompt-composer] { display: grid; gap: var(--space-xs); min-inline-size: 0; color: var(--color-foreground); padding-block-end: env(safe-area-inset-bottom, 0px); }
[data-rcl-composer-surface] { display: grid; min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); box-shadow: var(--elev-raised); transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard); }
[data-rcl-composer-surface]:focus-within, [data-rcl-composer-surface][data-dragging=true] { border-color: var(--color-primary); box-shadow: 0 0 0 var(--border-strong) color-mix(in srgb, var(--color-primary) 12%, transparent); }
[data-rcl-composer-surface] textarea[data-rcl-textarea] { border: 0; border-radius: var(--radius-panel); background: transparent; padding: var(--space-sm); min-block-size: 5rem; max-block-size: 15rem; resize: none; box-shadow: none; outline-offset: calc(var(--space-3xs) * -1); line-height: var(--text-body-line); }
[data-rcl-composer-toolbar] { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-2xs); padding: 0 var(--space-xs) var(--space-xs); }
[data-rcl-composer-tools] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-composer-caption] { margin: 0; color: var(--color-muted-foreground); font-size: var(--text-caption-size); line-height: var(--text-caption-line); overflow-wrap: anywhere; }
[data-rcl-composer-feedback] { margin: 0; min-block-size: 1.5em; font-size: var(--text-caption-size); color: var(--color-muted-foreground); }
[data-rcl-composer-feedback][data-error=true] { color: var(--color-danger); }
[data-rcl-composer-commands] { display: grid; gap: var(--space-3xs); padding: var(--space-2xs); border-block-end: var(--border-hairline) solid var(--color-border); }
[data-rcl-composer-context] { min-inline-size: 0; padding: var(--space-xs) var(--space-sm) 0; }
@media (pointer: coarse) { [data-rcl-composer-keyboard-hint] { display: none; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-composer-surface] { transition: none; } }
`;
const noSubscribe = () => () => {};
const emptySnapshot = () => 0;

export function PromptComposer({
  value,
  onValueChange,
  onSend,
  onSent,
  attachments = [],
  onFiles,
  onAttachmentRemove,
  onAttachmentRetry,
  onAttachmentCancel,
  onAttachmentReorder,
  commands,
  context,
  label = "Message",
  placeholder = "Write a message…",
  disabledReason,
  offline,
  generating,
  onStop,
  maxLength,
  accept,
  sendLabel = "Send message",
  className,
}: PromptComposerProps) {
  const id = useId();
  const textarea = useRef<HTMLTextAreaElement>(null);
  const submitButton = useRef<HTMLButtonElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const sending = useRef(false);
  const latestValue = useRef(value);
  latestValue.current = value;
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [dismissedCommand, setDismissedCommand] = useState<string | null>(null);
  useSyncExternalStore<unknown>(
    commands?.subscribe ?? noSubscribe,
    commands?.getSnapshot ?? emptySnapshot,
    commands?.getSnapshot ?? emptySnapshot,
  );
  const choices =
    commands && value.startsWith("/") && !value.includes("\n") && dismissedCommand !== value
      ? commands.search(value.slice(1)).slice(0, 5)
      : [];
  const unavailable =
    disabledReason ||
    (offline ? "You’re offline. Your draft is saved here; reconnect to send." : "");
  const waitingAttachments = attachments.some((item) => item.status && item.status !== "success");
  const tooLong = maxLength !== undefined && value.length > maxLength;
  const cannotSend = Boolean(
    unavailable ||
      pending ||
      generating ||
      tooLong ||
      waitingAttachments ||
      (!value.trim() && attachments.length === 0),
  );
  useLayoutEffect(() => {
    const element = textarea.current;
    if (!element) return;
    element.style.height = "auto";
    element.style.height = `${element.scrollHeight}px`;
  }, [value]);
  const send = async () => {
    if (cannotSend || sending.current) return;
    sending.current = true;
    setPending(true);
    setError("");
    setSent(false);
    const submission = { text: value, attachmentIds: attachments.map((item) => item.id) };
    try {
      if (!(await onSend(submission))) throw new Error("Message was not accepted.");
      if (latestValue.current === submission.text) onValueChange("");
      onSent?.(submission);
      setSent(true);
    } catch (cause) {
      setError("Couldn’t send. Your draft and attachments are still here. Try again.");
      throw cause;
    } finally {
      sending.current = false;
      setPending(false);
    }
  };
  const feedback =
    error ||
    unavailable ||
    (tooLong
      ? `Message exceeds the ${maxLength} character limit.`
      : waitingAttachments
        ? "Resolve unfinished attachments before sending."
        : pending
          ? "Sending message…"
          : sent
            ? "Message sent."
            : "");
  return (
    <form
      data-rcl-prompt-composer
      className={className}
      aria-label={`${label} composer`}
      onSubmit={(event) => {
        event.preventDefault();
        submitButton.current?.click();
      }}
    >
      <StyleSheet name="prompt-composer-1" css={css} />
      {attachments.length > 0 && (
        <ComposerAttachmentTray
          items={attachments}
          disabled={pending}
          onRemove={onAttachmentRemove}
          onRetry={onAttachmentRetry}
          onCancel={onAttachmentCancel}
          onReorder={onAttachmentReorder}
        />
      )}
      <div
        data-rcl-composer-surface
        data-dragging={dragging}
        onDragOver={(event) => {
          if (onFiles && !pending && event.dataTransfer.types.includes("Files")) {
            event.preventDefault();
            setDragging(true);
          }
        }}
        onDragLeave={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDragging(false);
        }}
        onDrop={(event) => {
          if (!onFiles || !event.dataTransfer.types.includes("Files")) return;
          event.preventDefault();
          setDragging(false);
          if (!pending) onFiles(Array.from(event.dataTransfer.files));
        }}
      >
        {context && <div data-rcl-composer-context>{context}</div>}
        {choices.length > 0 && (
          <div data-rcl-composer-commands role="group" aria-label="Suggested commands">
            {choices.map((command) => (
              <CommandButton
                key={command.id}
                type="button"
                variant="ghost"
                disabled={command.disabled || pending}
                action={() => commands!.execute(command.id, { query: value.slice(1) })}
                onSuccess={() => {
                  setDismissedCommand(value);
                  textarea.current?.focus();
                }}
                onError={() => setError("Command failed. Your draft is still here.")}
              >
                {command.label}
              </CommandButton>
            ))}
          </div>
        )}
        <Textarea
          ref={textarea}
          aria-label={label}
          aria-describedby={`${id}-hint ${id}-feedback`}
          aria-invalid={tooLong || undefined}
          value={value}
          placeholder={placeholder}
          onChange={(event) => {
            onValueChange(event.target.value);
            setSent(false);
          }}
          onPaste={(event) => {
            if (onFiles && !pending && event.clipboardData.files.length) {
              event.preventDefault();
              onFiles(Array.from(event.clipboardData.files));
            }
          }}
          onKeyDown={(event) => {
            if (event.key === "Escape" && choices.length) {
              event.preventDefault();
              setDismissedCommand(value);
            }
            if (
              event.key === "Enter" &&
              !event.shiftKey &&
              !event.altKey &&
              !event.nativeEvent.isComposing &&
              event.nativeEvent.keyCode !== 229
            ) {
              event.preventDefault();
              if (!cannotSend) submitButton.current?.click();
            }
          }}
        />
        <div data-rcl-composer-toolbar>
          <div data-rcl-composer-tools>
            {onFiles && (
              <>
                <input
                  ref={fileInput}
                  type="file"
                  multiple
                  accept={accept}
                  hidden
                  onChange={(event) => {
                    if (event.target.files?.length) onFiles(Array.from(event.target.files));
                    event.target.value = "";
                  }}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={pending}
                  onClick={() => fileInput.current?.click()}
                >
                  Attach files
                </Button>
              </>
            )}
            <p id={`${id}-hint`} data-rcl-composer-caption>
              <span data-rcl-composer-keyboard-hint>
                Enter to send · Shift+Enter for a new line
              </span>
              {maxLength !== undefined && (
                <span>
                  {" "}
                  {value.length}/{maxLength}
                </span>
              )}
            </p>
          </div>
          {generating && onStop ? (
            <CommandButton
              type="button"
              variant="secondary"
              action={onStop}
              pendingLabel="Stopping…"
            >
              Stop response
            </CommandButton>
          ) : (
            <CommandButton
              ref={submitButton}
              type="button"
              disabled={cannotSend}
              action={send}
              pendingLabel="Sending…"
              successLabel={sendLabel}
              errorLabel="Retry send"
            >
              {sendLabel}
            </CommandButton>
          )}
        </div>
      </div>
      <p
        id={`${id}-feedback`}
        data-rcl-composer-feedback
        data-error={Boolean(error || tooLong)}
        role="status"
        aria-live="polite"
      >
        {feedback}
      </p>
    </form>
  );
}
