/**
 * Shows the prompt a message would actually send, before sending it.
 *
 * The text is assembled by the server, which owns prompt construction. This
 * component never rebuilds it: a client that reimplemented the section order or
 * the volatility gradient would show a preview that agrees with nothing, which
 * is worse than showing none at all.
 */
import { useState } from "react";
import { Eye, Loader2 } from "lucide-react";
import { agentSessionService } from "../../services/agent-session-service";
import type { AgentSessionContextRef } from "../../types";
import { Dialog } from "../ui/dialog";

interface SessionPromptPreviewProps {
  sessionId: string;
  message: string;
  attachmentIds?: string[];
  contextRefs?: AgentSessionContextRef[];
  disabled?: boolean;
}

export function SessionPromptPreview({
  sessionId,
  message,
  attachmentIds,
  contextRefs,
  disabled,
}: SessionPromptPreviewProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [initial, setInitial] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const open = async () => {
    setIsOpen(true);
    setLoading(true);
    setError(null);
    try {
      const preview = await agentSessionService.previewPrompt({
        sessionId,
        message,
        attachmentIds,
        contextRefs,
      });
      setPrompt(preview.prompt);
      setInitial(preview.initial);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not assemble the prompt preview.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={open}
        disabled={disabled}
        title="Show the prompt this message would send"
        className="inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 disabled:cursor-not-allowed disabled:opacity-50"
        data-testid="session-prompt-preview-open"
      >
        <Eye className="h-3.5 w-3.5" />
        Preview prompt
      </button>

      <Dialog
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="Prompt preview"
        maxWidth="max-w-3xl"
        testId="session-prompt-preview"
      >
        <p className="mb-3 text-xs text-slate-400">
          {initial
            ? "The full initial prompt, including the stable doctrine bands and the startup brief."
            : "The continuation prompt. Stable bands are omitted — they are already in the conversation."}
        </p>
        {loading && (
          <p className="flex items-center gap-2 text-sm text-slate-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Assembling…
          </p>
        )}
        {error && <p className="text-sm text-red-300">{error}</p>}
        {!loading && !error && (
          <>
            <p className="mb-2 text-xs text-slate-500">
              {prompt.length.toLocaleString()} characters · assembled by the server, exactly as the agent receives it
            </p>
            <pre
              className="max-h-[60vh] overflow-auto whitespace-pre-wrap break-words rounded border border-slate-800 bg-slate-950/70 p-3 font-mono text-[11px] leading-relaxed text-slate-300"
              data-testid="session-prompt-preview-text"
            >
              {prompt}
            </pre>
          </>
        )}
      </Dialog>
    </>
  );
}
