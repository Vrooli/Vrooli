/**
 * PlanPanel Component
 *
 * Displays the backlog item's plan.md content with:
 * - Rendered markdown view (default)
 * - Raw Monaco editor for editing
 * - Save-to-API for persisting edits
 * - Copy-to-clipboard
 * - Table of contents popover for heading navigation
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type OnMount } from "@monaco-editor/react";
import {
  Check,
  Code,
  Copy,
  Eye,
  FileText,
  List,
  Loader2,
  Save,
} from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, useResolvedTheme } from "../../lib";
import { renderMarkdown } from "../../lib/render-markdown";
import { extractHeadings, type HeadingEntry } from "../../lib/heading-utils";
import { backlogService } from "../../services";
import type { BacklogKind } from "../../types";
import { getDeliverablePath } from "../../lib/workshop-files";
import { useModalBehavior } from "../../hooks/useModalBehavior";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { selectors } from "../../consts/selectors";

export interface PlanPanelProps {
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

const TOC_ITEM_STYLES: Record<number, string> = {
  1: "pl-3 font-medium text-slate-200",
  2: "pl-5 text-slate-400",
  3: "pl-8 text-slate-500 text-xs",
};

export function PlanPanel({ backlogKind, backlogName, className }: PlanPanelProps) {
  const queryClient = useQueryClient();
  const resolvedTheme = useResolvedTheme();

  const deliverablePath = getDeliverablePath(backlogKind);
  const isResearch = backlogKind === "research";
  const deliverableLabel = isResearch ? "conclusion" : "plan";

  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");
  const [draftContent, setDraftContent] = useState("");
  const [copySuccess, setCopySuccess] = useState(false);
  const [tocOpen, setTocOpen] = useState(false);

  const tocRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);

  useModalBehavior({
    isOpen: tocOpen,
    onClose: () => setTocOpen(false),
    ref: tocRef,
    delayClickOutside: true,
  });

  const queryKey = ["backlog-plan-content", backlogKind, backlogName];

  const {
    data: planContent,
    isLoading,
    error,
    refetch,
  } = useQuery<string>({
    queryKey,
    queryFn: () => backlogService.getFileContent(backlogKind, backlogName, deliverablePath),
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
      backlogService.saveFileContent(backlogKind, backlogName, deliverablePath, content),
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

  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor;
  }, []);

  const handleTocToggle = useCallback(() => {
    setTocOpen((prev) => !prev);
  }, []);

  const handleTocJump = useCallback((heading: HeadingEntry) => {
    setTocOpen(false);
    if (viewMode === "rendered") {
      document.getElementById(heading.id)?.scrollIntoView({ behavior: "smooth", block: "start" });
    } else if (editorRef.current) {
      editorRef.current.revealLineInCenter(heading.line);
      editorRef.current.setPosition({ lineNumber: heading.line, column: 1 });
      editorRef.current.focus();
    }
  }, [viewMode]);

  const headings = extractHeadings(draftContent);

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
        <p className="text-sm font-medium text-slate-400">No {deliverableLabel} yet</p>
        <p className="text-xs text-slate-500">Run a workshop session to generate a {deliverableLabel}.</p>
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
          size="icon"
          className={cn(viewMode === "rendered" && "bg-slate-700 text-slate-100")}
          onClick={() => setViewMode("rendered")}
          aria-label="Rendered view"
          title="Rendered view"
        >
          <Eye className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          className={cn(viewMode === "raw" && "bg-slate-700 text-slate-100")}
          onClick={() => setViewMode("raw")}
          aria-label="Raw editor"
          title="Edit"
        >
          <Code className="h-3.5 w-3.5" />
        </Button>

        {headings.length > 0 && (
          <div ref={tocRef} className="relative">
            <Button
              variant="outline"
              size="icon"
              className={cn(tocOpen && "bg-slate-700 text-slate-100")}
              onClick={handleTocToggle}
              aria-label="Table of contents"
              title="Table of contents"
            >
              <List className="h-3.5 w-3.5" />
            </Button>
            {tocOpen && (
              <nav
                className="absolute left-0 top-full z-50 mt-1 w-56 overflow-hidden rounded-lg border border-white/10 bg-slate-900/95 shadow-xl backdrop-blur-sm animate-in fade-in-0 zoom-in-95 duration-100"
                aria-label="Table of contents"
                data-testid="toc-popover"
              >
                <div className="max-h-72 overflow-y-auto py-1.5">
                  {headings.map((h, i) => {
                    const showDivider = h.level === 1 && i > 0;
                    return (
                      <div key={`${h.id}-${h.line}`}>
                        {showDivider && (
                          <div className="mx-3 my-1 border-t border-white/5" />
                        )}
                        <button
                          className={cn(
                            "block w-full truncate py-1 pr-3 text-left text-[13px] transition-colors",
                            "hover:bg-white/5 hover:text-slate-100",
                            TOC_ITEM_STYLES[h.level],
                          )}
                          onClick={() => handleTocJump(h)}
                        >
                          {h.text}
                        </button>
                      </div>
                    );
                  })}
                </div>
              </nav>
            )}
          </div>
        )}

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
          size="icon"
          className={cn(copySuccess && "text-green-400")}
          onClick={handleCopy}
          aria-label={`Copy ${deliverableLabel}`}
          title={copySuccess ? "Copied!" : `Copy ${deliverableLabel}`}
        >
          {copySuccess ? (
            <Check className="h-3.5 w-3.5" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </Button>
      </div>

      {/* Save feedback */}
      {saveMutation.isSuccess && (
        <div className="border-b border-green-500/20 bg-green-500/10 px-4 py-1.5 text-xs text-green-400">
          {isResearch ? "Conclusion" : "Plan"} saved.
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
            onMount={handleEditorMount}
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
