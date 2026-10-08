/**
 * ExecutionPromptTab — Displays the prompt trace for an execution,
 * including purpose, prompt text, revision, and metadata.
 */

import { Loader2, Sparkles } from "lucide-react";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { PromptTrace } from "../../types";

export interface ExecutionPromptTabProps {
  trace: PromptTrace | null | undefined;
  isLoading: boolean;
}

export function ExecutionPromptTab({ trace, isLoading }: ExecutionPromptTabProps) {
  if (isLoading) {
    return (
      <DetailSection title="Prompt Trace" hideDivider>
        <div className="flex items-center gap-2 py-6 justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
          <span className="text-sm text-slate-400">Loading prompt trace...</span>
        </div>
      </DetailSection>
    );
  }

  if (!trace) {
    return (
      <DetailSection title="Prompt Trace" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.promptEmpty}>
          <Sparkles className="mx-auto mb-2 h-8 w-8 text-slate-600" />
          <p className="text-sm text-slate-400">No prompt trace available for this execution.</p>
        </div>
      </DetailSection>
    );
  }

  return (
    <DetailSection title="Prompt Trace" hideDivider>
      <div className="space-y-3" data-testid={selectors.executionDetails.promptTrace}>
        <div>
          <p className="text-xs text-slate-500 uppercase tracking-wider">Purpose</p>
          <p className="text-sm text-slate-200">{trace.purpose}</p>
        </div>
        <div>
          <p className="text-xs text-slate-500 uppercase tracking-wider">Prompt</p>
          <pre className="mt-1 max-h-96 overflow-auto rounded-lg bg-slate-800/60 p-3 text-xs text-slate-300 whitespace-pre-wrap">
            {trace.prompt}
          </pre>
        </div>
        {trace.prompt_revision && (
          <div>
            <p className="text-xs text-slate-500 uppercase tracking-wider">Revision</p>
            <pre className="mt-1 max-h-96 overflow-auto rounded-lg bg-slate-800/60 p-3 text-xs text-slate-300 whitespace-pre-wrap">
              {trace.prompt_revision}
            </pre>
          </div>
        )}
        <div className="flex items-center gap-4 text-xs text-slate-400">
          <span>Captured: {trace.captured_at}</span>
          {trace.used_fallback && (
            <span className="rounded bg-amber-500/20 px-2 py-0.5 text-amber-300">Fallback used</span>
          )}
          {trace.synthetic && (
            <span
              className="rounded bg-sky-500/20 px-2 py-0.5 text-sky-300"
              title="Reconstructed caller context — the agent ran the bound operation's mode prompt, not this text verbatim."
            >
              Reconstructed context
            </span>
          )}
        </div>
      </div>
    </DetailSection>
  );
}
