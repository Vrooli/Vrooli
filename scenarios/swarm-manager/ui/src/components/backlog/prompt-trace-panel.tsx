/**
 * PromptTracePanel Component
 *
 * Displays the generated prompt trace for a backlog item with:
 * - Rendered markdown view (default)
 * - Raw Monaco editor for editing
 * - Save-to-API for persisting edits
 * - Metadata bar (skill_id, captured_at, used_fallback)
 * - Copy-to-clipboard
 */

import { useCallback, useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor from "@monaco-editor/react";
import {
  Check,
  Code,
  Copy,
  Eye,
  Loader2,
  Save,
  Sparkles,
  TriangleAlert,
} from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, formatRelativeTime, useResolvedTheme } from "../../lib";
import { renderMarkdown } from "../../lib/render-markdown";
import { promptService } from "../../services";
import type { BacklogKind, PromptTrace } from "../../types";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { selectors } from "../../consts/selectors";

export interface PromptTracePanelProps {
  backlogKind: BacklogKind;
  backlogName: string;
  className?: string;
}

const EDITOR_OPTIONS = {
  minimap: { enabled: false },
  wordWrap: "on",
  lineNumbers: "on",
  fontSize: 13,
  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
  tabSize: 2,
  scrollBeyondLastLine: false,
  renderLineHighlight: "none",
  cursorBlinking: "smooth",
  smoothScrolling: true,
  scrollbar: {
    vertical: "auto",
    horizontal: "auto",
    verticalScrollbarSize: 8,
    horizontalScrollbarSize: 8,
  },
  hideCursorInOverviewRuler: true,
  folding: true,
  foldingStrategy: "indentation",
  automaticLayout: true,
} as const;

export function PromptTracePanel({ backlogKind, backlogName, className }: PromptTracePanelProps) {
  const queryClient = useQueryClient();
  const resolvedTheme = useResolvedTheme();

  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");
  const [draftPrompt, setDraftPrompt] = useState("");
  const [copySuccess, setCopySuccess] = useState(false);

  const queryKey = ["backlog-prompt-trace", backlogKind, backlogName];

  const {
    data: trace,
    isLoading,
    error,
    refetch,
  } = useQuery<PromptTrace>({
    queryKey,
    queryFn: () => promptService.getBacklogPromptTrace(backlogKind, backlogName),
    ...defaultQueryOptions,
  });

  // Sync draft when trace loads or changes
  useEffect(() => {
    if (trace) {
      setDraftPrompt(trace.prompt);
    }
  }, [trace]);

  const isDirty = trace ? draftPrompt !== trace.prompt : false;

  const saveMutation = useMutation({
    mutationFn: (updatedPrompt: string) => {
      if (!trace) throw new Error("No trace to update");
      return promptService.updateBacklogPromptTrace(backlogKind, backlogName, {
        ...trace,
        prompt: updatedPrompt,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const handleCopy = useCallback(async () => {
    if (!trace) return;
    await navigator.clipboard.writeText(trace.prompt);
    setCopySuccess(true);
    setTimeout(() => setCopySuccess(false), 2000);
  }, [trace]);

  const handleSave = useCallback(() => {
    if (isDirty) {
      saveMutation.mutate(draftPrompt);
    }
  }, [isDirty, draftPrompt, saveMutation]);

  const handleDiscard = useCallback(() => {
    if (trace) {
      setDraftPrompt(trace.prompt);
    }
  }, [trace]);

  // Detect 404 (no trace yet)
  const is404 = error && (error as Error).message?.includes("not found");

  if (isLoading) {
    return (
      <div className={cn("flex items-center justify-center py-20", className)}>
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  if (is404 || (!isLoading && !error && !trace)) {
    return (
      <div
        className={cn("flex flex-col items-center justify-center gap-3 py-20 text-center", className)}
        data-testid={selectors.backlogDetails.promptPanel}
      >
        <Sparkles className="h-10 w-10 text-slate-600" />
        <p className="text-sm font-medium text-slate-400">No prompt trace yet</p>
        <p className="text-xs text-slate-500">Run a workshop session to generate a prompt trace.</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className={cn("px-4 py-8", className)}>
        <ErrorState error={error as Error} onRetry={() => refetch()} />
      </div>
    );
  }

  if (!trace) return null;

  return (
    <div
      className={cn("flex flex-col", className)}
      data-testid={selectors.backlogDetails.promptPanel}
    >
      {/* Metadata bar */}
      <div className="flex flex-wrap items-center gap-2 border-b border-slate-800 px-4 py-2.5">
        <span className="rounded-full bg-slate-700 px-2.5 py-0.5 text-xs font-medium text-slate-300">
          {trace.skill_id}
        </span>
        <span className="text-xs text-slate-500">
          {formatRelativeTime(trace.captured_at)}
        </span>
        {trace.used_fallback && (
          <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-400">
            <TriangleAlert className="h-3 w-3" />
            Fallback
          </span>
        )}
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-1.5 border-b border-slate-800 px-4 py-2">
        <Button
          variant="outline"
          size="sm"
          className={cn(
            "h-8 px-2.5 text-xs",
            viewMode === "rendered" && "bg-slate-700 text-slate-100",
          )}
          onClick={() => setViewMode("rendered")}
          aria-label="Rendered view"
        >
          <Eye className="mr-1.5 h-3.5 w-3.5" />
          Rendered
        </Button>
        <Button
          variant="outline"
          size="sm"
          className={cn(
            "h-8 px-2.5 text-xs",
            viewMode === "raw" && "bg-slate-700 text-slate-100",
          )}
          onClick={() => setViewMode("raw")}
          aria-label="Raw editor"
        >
          <Code className="mr-1.5 h-3.5 w-3.5" />
          Edit
        </Button>

        <div className="flex-1" />

        {viewMode === "raw" && isDirty && (
          <Button
            variant="outline"
            size="sm"
            className="h-8 px-2.5 text-xs text-slate-400"
            onClick={handleDiscard}
          >
            Discard
          </Button>
        )}

        {viewMode === "raw" && (
          <Button
            variant="outline"
            size="sm"
            className="h-8 px-2.5 text-xs"
            disabled={!isDirty || saveMutation.isPending}
            onClick={handleSave}
          >
            {saveMutation.isPending ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="mr-1.5 h-3.5 w-3.5" />
            )}
            Save
          </Button>
        )}

        <Button
          variant="outline"
          size="sm"
          className="h-8 px-2.5 text-xs"
          onClick={handleCopy}
          aria-label="Copy prompt"
        >
          {copySuccess ? (
            <Check className="mr-1.5 h-3.5 w-3.5 text-green-400" />
          ) : (
            <Copy className="mr-1.5 h-3.5 w-3.5" />
          )}
          {copySuccess ? "Copied" : "Copy"}
        </Button>
      </div>

      {/* Save feedback */}
      {saveMutation.isSuccess && (
        <div className="border-b border-green-500/20 bg-green-500/10 px-4 py-1.5 text-xs text-green-400">
          Prompt trace saved.
        </div>
      )}
      {saveMutation.isError && (
        <div className="border-b border-red-500/20 bg-red-500/10 px-4 py-1.5 text-xs text-red-400">
          Failed to save: {(saveMutation.error as Error).message}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {viewMode === "rendered" ? (
          <div
            className="px-4 py-4"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(trace.prompt) }}
          />
        ) : (
          <Editor
            height="100%"
            language="markdown"
            theme={resolvedTheme === "dark" ? "vs-dark" : "light"}
            value={draftPrompt}
            onChange={(value) => setDraftPrompt(value ?? "")}
            options={EDITOR_OPTIONS}
            loading={
              <div className="flex items-center justify-center py-20">
                <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
              </div>
            }
          />
        )}
      </div>
    </div>
  );
}
