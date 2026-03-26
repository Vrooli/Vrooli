/**
 * PlanPanel Component
 *
 * Displays the backlog item's plan.md content with:
 * - Rendered markdown view (default)
 * - Raw Monaco editor for editing
 * - Save-to-API for persisting edits
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
  FileText,
  Loader2,
  Save,
} from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, useResolvedTheme } from "../../lib";
import { renderMarkdown } from "../../lib/render-markdown";
import { backlogService } from "../../services";
import type { BacklogKind } from "../../types";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { selectors } from "../../consts/selectors";

export interface PlanPanelProps {
  backlogKind: BacklogKind;
  backlogName: string;
  className?: string;
}

const PLAN_FILE_PATH = "plan.md";

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

export function PlanPanel({ backlogKind, backlogName, className }: PlanPanelProps) {
  const queryClient = useQueryClient();
  const resolvedTheme = useResolvedTheme();

  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");
  const [draftContent, setDraftContent] = useState("");
  const [copySuccess, setCopySuccess] = useState(false);

  const queryKey = ["backlog-plan-content", backlogKind, backlogName];

  const {
    data: planContent,
    isLoading,
    error,
    refetch,
  } = useQuery<string>({
    queryKey,
    queryFn: () => backlogService.getFileContent(backlogKind, backlogName, PLAN_FILE_PATH),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    if (planContent !== undefined) {
      setDraftContent(planContent);
    }
  }, [planContent]);

  const isDirty = planContent !== undefined ? draftContent !== planContent : false;

  const saveMutation = useMutation({
    mutationFn: (content: string) =>
      backlogService.saveFileContent(backlogKind, backlogName, PLAN_FILE_PATH, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const handleCopy = useCallback(async () => {
    if (planContent === undefined) return;
    await navigator.clipboard.writeText(draftContent);
    setCopySuccess(true);
    setTimeout(() => setCopySuccess(false), 2000);
  }, [planContent, draftContent]);

  const handleSave = useCallback(() => {
    if (isDirty) {
      saveMutation.mutate(draftContent);
    }
  }, [isDirty, draftContent, saveMutation]);

  const handleDiscard = useCallback(() => {
    if (planContent !== undefined) {
      setDraftContent(planContent);
    }
  }, [planContent]);

  const is404 = error && (error as Error).message?.includes("not found");

  if (isLoading) {
    return (
      <div className={cn("flex items-center justify-center py-20", className)}>
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  if (is404 || (!isLoading && !error && planContent === undefined)) {
    return (
      <div
        className={cn("flex flex-col items-center justify-center gap-3 py-20 text-center", className)}
        data-testid={selectors.backlogDetails.promptPanel}
      >
        <FileText className="h-10 w-10 text-slate-600" />
        <p className="text-sm font-medium text-slate-400">No plan yet</p>
        <p className="text-xs text-slate-500">Run a workshop session to generate a plan.</p>
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

  if (planContent === undefined) return null;

  return (
    <div
      className={cn("flex flex-col", className)}
      data-testid={selectors.backlogDetails.promptPanel}
    >
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
          aria-label="Copy plan"
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
          Plan saved.
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
            dangerouslySetInnerHTML={{ __html: renderMarkdown(planContent) }}
          />
        ) : (
          <Editor
            height="100%"
            language="markdown"
            theme={resolvedTheme === "dark" ? "vs-dark" : "light"}
            value={draftContent}
            onChange={(value) => setDraftContent(value ?? "")}
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
