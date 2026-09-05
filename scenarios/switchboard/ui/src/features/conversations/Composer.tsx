import { Paperclip, SendHorizontal, X } from "lucide-react";
import { useRef, useState, type FormEvent, type KeyboardEvent } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatBytes } from "./Transcript";

interface ComposerProps {
  /** When set the composer renders disabled with this reason instead of a textbox. */
  disabledReason?: string;
  busy?: boolean;
  onSend: (text: string, attachment?: File) => boolean | Promise<boolean>;
}

/**
 * The message composer. Anchored at the bottom, grows with its text, sends on
 * Enter (Shift+Enter for a newline), and never loses a draft on a failed send.
 * A disabled composer always says why.
 */
export function Composer({ disabledReason, busy, onSend }: ComposerProps) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [attachment, setAttachment] = useState<File>();
  const [error, setError] = useState<string>();
  const fileInput = useRef<HTMLInputElement>(null);
  const canSend = !disabledReason && !busy && (text.trim().length > 0 || attachment !== undefined);

  const submit = async (event?: FormEvent) => {
    event?.preventDefault();
    if (!canSend) return;
    setError(undefined);
    const ok = await onSend(text.trim(), attachment);
    if (ok) {
      setText("");
      setAttachment(undefined);
    } else {
      setError(t(strings.console.conversations.sendFailed));
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void submit();
    }
  };

  if (disabledReason) {
    return (
      <div data-testid="conversations-composer" role="status" className="rounded-panel border border-dashed border-app-border bg-app-surface px-4 py-3 text-sm text-app-muted-foreground">
        {disabledReason}
      </div>
    );
  }

  return (
    <form
      data-testid="conversations-composer"
      aria-label={t(strings.console.conversations.sendLabel)}
      onSubmit={(event) => void submit(event)}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-2 shadow-subtle focus-within:border-app-primary"
    >
      {attachment ? (
        <div className="flex items-center gap-2 px-1 text-xs">
          <Paperclip aria-hidden="true" className="h-3.5 w-3.5 text-app-muted-foreground" />
          <span className="truncate font-medium">{attachment.name}</span>
          <span className="text-app-muted-foreground">{formatBytes(attachment.size)}</span>
          <button type="button" aria-label={t(strings.console.conversations.removeAttachment)} onClick={() => setAttachment(undefined)} className="ml-auto grid h-11 w-11 place-items-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted md:h-7 md:w-7">
            <X aria-hidden="true" className="h-3.5 w-3.5" />
          </button>
        </div>
      ) : null}
      <div className="flex items-end gap-2">
        <label className="sr-only" htmlFor="conversation-input">
          {t(strings.console.conversations.message)}
        </label>
        <textarea
          id="conversation-input"
          data-testid="conversations-composer-input"
          value={text}
          rows={1}
          onChange={(event) => setText(event.target.value)}
          onKeyDown={onKeyDown}
          placeholder={t(strings.console.conversations.writePlaceholder)}
          disabled={busy}
          className="max-h-40 min-h-10 flex-1 resize-none bg-transparent px-2 py-2 text-base leading-6 text-app-foreground placeholder:text-app-muted-foreground focus:outline-none md:text-sm"
          style={{ height: `${Math.min(160, 24 * Math.max(1, text.split("\n").length) + 16)}px` }}
        />
        <input
          ref={fileInput}
          type="file"
          className="sr-only"
          tabIndex={-1}
          onChange={(event) => {
            setAttachment(event.target.files?.[0]);
            event.target.value = "";
          }}
        />
        <Button type="button" variant="ghost" size="icon" className="min-h-11 min-w-11 md:min-h-10 md:min-w-10" data-testid="conversations-attach" aria-label={t(strings.console.conversations.attachFile)} onClick={() => fileInput.current?.click()} disabled={busy}>
          <Paperclip aria-hidden="true" className="h-4 w-4" />
        </Button>
        <Button type="submit" size="icon" className="min-h-11 min-w-11 md:min-h-10 md:min-w-10" data-testid="conversations-send" aria-label={t(strings.console.conversations.send)} disabled={!canSend} pending={busy}>
          <SendHorizontal aria-hidden="true" className="h-4 w-4" />
        </Button>
      </div>
      {error ? (
        <p role="alert" className="px-1 text-xs text-app-danger">
          {error}
        </p>
      ) : (
        <p className="hidden px-1 text-[11px] text-app-muted-foreground md:block">{t(strings.console.conversations.enterHint)}</p>
      )}
    </form>
  );
}
