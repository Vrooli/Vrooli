import { useState, useEffect, useCallback, useRef } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Copy,
  Check,
  Download,
  Upload,
  ChevronDown,
  ChevronRight,
  MessageSquareText,
  Sparkles,
  Info,
} from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { Select } from "../ui/select";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../ui/tabs";
import { selectors } from "../../consts/selectors";
import { backlogService } from "../../services";
import type { ImportBacklogResponse } from "../../services/backlog-service";
import {
  IDEA_AGENT_FILE_PATHS,
  parseClarifyQuestionsFile,
  buildClarifyQuestionsContent,
  parseSuggestionsFile,
  buildSuggestionsContent,
} from "../../lib";
import type {
  BacklogKind,
  BacklogStatus,
  FeedbackSummaryResponse,
  FeedbackItemSummary,
  IdeaClarificationQuestion,
  IdeaSuggestion,
  IdeaSuggestionDecision,
} from "../../types";

type TabKind = BacklogKind | "all";

interface FeedbackHubModalProps {
  isOpen: boolean;
  onClose: () => void;
  feedbackSummary: FeedbackSummaryResponse | undefined;
  activeKind: TabKind;
  statusFilter: BacklogStatus | "";
  selectedNames?: string[];
  onDataChanged: () => void;
  initialTab?: "review" | "export" | "import";
}

const DECISION_OPTIONS: Array<{ value: IdeaSuggestionDecision; label: string }> = [
  { value: "accepted", label: "Accept" },
  { value: "rejected", label: "Reject" },
  { value: "pending", label: "Pending" },
];

// ---------------------------------------------------------------------------
// Review Tab - Expandable item sections with inline question/suggestion editing
// ---------------------------------------------------------------------------

interface ReviewItemState {
  questions: IdeaClarificationQuestion[];
  questionsRaw: Record<string, unknown> | null;
  suggestions: IdeaSuggestion[];
  suggestionsRaw: Record<string, unknown> | null;
  saved: boolean;
}

function ReviewItemSection({
  item,
  onSaved,
}: {
  item: FeedbackItemSummary;
  onSaved: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [local, setLocal] = useState<ReviewItemState | null>(null);

  const clarifyQuery = useQuery({
    queryKey: ["feedback-hub-clarify", item.kind, item.name],
    queryFn: () =>
      backlogService.getFileContent(item.kind, item.name, IDEA_AGENT_FILE_PATHS.clarify),
    enabled: expanded,
  });

  const suggestQuery = useQuery({
    queryKey: ["feedback-hub-suggest", item.kind, item.name],
    queryFn: () =>
      backlogService.getFileContent(item.kind, item.name, IDEA_AGENT_FILE_PATHS.suggest),
    enabled: expanded,
  });

  useEffect(() => {
    if (!expanded) return;
    const cResult = parseClarifyQuestionsFile(clarifyQuery.data);
    const sResult = parseSuggestionsFile(suggestQuery.data);
    setLocal({
      questions: cResult.questions,
      questionsRaw: cResult.raw,
      suggestions: sResult.suggestions,
      suggestionsRaw: sResult.raw,
      saved: false,
    });
  }, [expanded, clarifyQuery.data, suggestQuery.data]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!local) return;
      await Promise.all([
        local.questions.length > 0
          ? backlogService.saveFileContent(
              item.kind,
              item.name,
              IDEA_AGENT_FILE_PATHS.clarify,
              buildClarifyQuestionsContent(local.questionsRaw, local.questions),
              "application/json",
            )
          : Promise.resolve(),
        local.suggestions.length > 0
          ? backlogService.saveFileContent(
              item.kind,
              item.name,
              IDEA_AGENT_FILE_PATHS.suggest,
              buildSuggestionsContent(local.suggestionsRaw, local.suggestions),
              "application/json",
            )
          : Promise.resolve(),
      ]);
    },
    onSuccess: () => {
      setLocal((prev) => (prev ? { ...prev, saved: true } : prev));
      onSaved();
    },
  });

  const counts: string[] = [];
  if (item.unanswered_questions > 0)
    counts.push(`${item.unanswered_questions} question${item.unanswered_questions === 1 ? "" : "s"}`);
  if (item.pending_suggestions > 0)
    counts.push(`${item.pending_suggestions} suggestion${item.pending_suggestions === 1 ? "" : "s"}`);

  const isLoading = clarifyQuery.isLoading || suggestQuery.isLoading;

  return (
    <div className="rounded-lg border border-white/10 bg-slate-800/40">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center gap-2 px-4 py-3 text-left text-sm font-medium text-slate-100 hover:bg-slate-800/60 transition-colors"
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 shrink-0 text-slate-400" />
        ) : (
          <ChevronRight className="h-4 w-4 shrink-0 text-slate-400" />
        )}
        <span className="truncate">{item.title || item.name}</span>
        <span className="ml-1 rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
          {item.kind}
        </span>
        {counts.length > 0 && (
          <span className="ml-auto text-xs text-slate-400">{counts.join(", ")}</span>
        )}
      </button>

      {expanded && (
        <div className="border-t border-white/5 px-4 py-4 space-y-4">
          {isLoading && (
            <div className="flex items-center gap-2 py-4">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-500 border-t-cyan-400" />
              <span className="text-sm text-slate-400">Loading feedback...</span>
            </div>
          )}

          {!isLoading && local && (
            <>
              {/* Questions */}
              {local.questions.length > 0 && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
                    <MessageSquareText className="h-4 w-4 text-cyan-400" />
                    Questions
                  </div>
                  {local.questions.map((q, idx) => (
                    <div key={q.id} className="rounded-lg border border-white/10 bg-slate-900/40 p-3">
                      <p className="text-sm text-slate-100">
                        {idx + 1}. {q.question}
                      </p>
                      <textarea
                        value={q.answer ?? ""}
                        onChange={(e) => {
                          const val = e.target.value;
                          setLocal((prev) =>
                            prev
                              ? {
                                  ...prev,
                                  saved: false,
                                  questions: prev.questions.map((item) =>
                                    item.id === q.id ? { ...item, answer: val } : item,
                                  ),
                                }
                              : prev,
                          );
                        }}
                        placeholder="Your answer..."
                        className="mt-2 w-full rounded-md border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                        rows={2}
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* Suggestions */}
              {local.suggestions.length > 0 && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
                    <Sparkles className="h-4 w-4 text-cyan-400" />
                    Suggestions
                  </div>
                  {local.suggestions.map((s, idx) => (
                    <div key={s.id} className="rounded-lg border border-white/10 bg-slate-900/40 p-3">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-medium text-slate-100">Suggestion {idx + 1}</p>
                        <Select
                          value={s.status ?? "pending"}
                          onChange={(e) => {
                            const status = e.target.value as IdeaSuggestionDecision;
                            setLocal((prev) =>
                              prev
                                ? {
                                    ...prev,
                                    saved: false,
                                    suggestions: prev.suggestions.map((item) =>
                                      item.id === s.id ? { ...item, status } : item,
                                    ),
                                  }
                                : prev,
                            );
                          }}
                          variant="compact"
                        >
                          {DECISION_OPTIONS.map((opt) => (
                            <option key={opt.value} value={opt.value}>
                              {opt.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <p className="mt-2 text-sm text-slate-300">{s.suggestion}</p>
                      {s.details && <p className="mt-1 text-xs text-slate-400">{s.details}</p>}
                    </div>
                  ))}
                </div>
              )}

              {saveMutation.isError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                  Failed to save. Please try again.
                </div>
              )}

              {local.saved && (
                <div className="flex items-center gap-2 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-2 text-sm text-green-300">
                  <Info className="h-4 w-4" />
                  Answers saved. Visit the item detail page to trigger the next agent step.
                </div>
              )}

              <div className="flex justify-end">
                <Button
                  size="sm"
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                >
                  {saveMutation.isPending ? "Saving..." : "Save"}
                </Button>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Export Tab
// ---------------------------------------------------------------------------

function ExportTab({
  activeKind,
  statusFilter,
  selectedNames,
}: {
  activeKind: TabKind;
  statusFilter: BacklogStatus | "";
  selectedNames?: string[];
}) {
  const [includePrd, setIncludePrd] = useState(false);
  const [includeRequirements, setIncludeRequirements] = useState(false);
  const [content, setContent] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const exportMutation = useMutation({
    mutationFn: async () => {
      const params: Parameters<typeof backlogService.exportItems>[0] = {
        includePrd,
        includeRequirements,
      };
      if (activeKind !== "all") params.kinds = [activeKind];
      if (statusFilter) params.statuses = [statusFilter];
      if (selectedNames && selectedNames.length > 0) params.names = selectedNames;
      const blob = await backlogService.exportItems(params);
      return blob.text();
    },
    onSuccess: (text) => setContent(text),
  });

  const handleCopy = async () => {
    if (!content) return;
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard not available
    }
  };

  const handleDownload = () => {
    if (!content) return;
    const blob = new Blob([content], { type: "text/markdown" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `backlog-export-${new Date().toISOString().slice(0, 10)}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-4" data-testid={selectors.backlog.feedbackHub.exportTab}>
      <div className="flex flex-wrap gap-4">
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input
            type="checkbox"
            checked={includePrd}
            onChange={(e) => setIncludePrd(e.target.checked)}
            className="rounded border-white/20 bg-slate-800"
          />
          Include PRD
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input
            type="checkbox"
            checked={includeRequirements}
            onChange={(e) => setIncludeRequirements(e.target.checked)}
            className="rounded border-white/20 bg-slate-800"
          />
          Include Requirements
        </label>
      </div>

      <div className="flex items-center gap-2 text-xs text-slate-400">
        {activeKind !== "all" && <span>Kind: {activeKind}</span>}
        {statusFilter && <span>Status: {statusFilter}</span>}
        {selectedNames && selectedNames.length > 0 && (
          <span>{selectedNames.length} item{selectedNames.length === 1 ? "" : "s"} selected</span>
        )}
      </div>

      <Button
        size="sm"
        onClick={() => {
          setContent(null);
          setCopied(false);
          exportMutation.mutate();
        }}
        disabled={exportMutation.isPending}
      >
        {exportMutation.isPending ? "Generating..." : "Generate Export"}
      </Button>

      {exportMutation.isError && (
        <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          Failed to generate export. Please try again.
        </div>
      )}

      {content !== null && (
        <>
          <pre className="max-h-[40vh] overflow-auto rounded-lg border border-slate-700/70 bg-slate-800/60 p-4 text-sm text-slate-300 font-mono whitespace-pre-wrap break-words">
            {content}
          </pre>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? (
                <Check className="mr-2 h-4 w-4 text-green-400" />
              ) : (
                <Copy className="mr-2 h-4 w-4" />
              )}
              {copied ? "Copied" : "Copy"}
            </Button>
            <Button size="sm" onClick={handleDownload}>
              <Download className="mr-2 h-4 w-4" />
              Download
            </Button>
          </div>
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Import Tab
// ---------------------------------------------------------------------------

function ImportTab({ onDataChanged }: { onDataChanged: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportBacklogResponse | null>(null);
  const [dragging, setDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const previewMutation = useMutation({
    mutationFn: (f: File) => backlogService.importItems(f, false),
    onSuccess: (data) => setPreview(data),
  });

  const applyMutation = useMutation({
    mutationFn: () => {
      if (!file) throw new Error("No file selected");
      return backlogService.importItems(file, true);
    },
    onSuccess: () => {
      onDataChanged();
      setFile(null);
      setPreview(null);
    },
  });

  const handleFile = useCallback((f: File) => {
    setFile(f);
    setPreview(null);
    previewMutation.mutate(f);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragging(false);
      const dropped = e.dataTransfer.files[0];
      if (dropped) handleFile(dropped);
    },
    [handleFile],
  );

  return (
    <div className="space-y-4" data-testid={selectors.backlog.feedbackHub.importTab}>
      {/* Drop zone */}
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
        className={`flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-8 transition-colors ${
          dragging
            ? "border-cyan-400 bg-cyan-500/10"
            : "border-white/10 bg-slate-800/40 hover:border-white/20"
        }`}
      >
        <Upload className="h-6 w-6 text-slate-400" />
        <p className="text-sm text-slate-300">
          {file ? file.name : "Drop a markdown file here or click to browse"}
        </p>
        <input
          ref={fileInputRef}
          type="file"
          accept=".md,.markdown,.txt"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleFile(f);
          }}
        />
      </div>

      {previewMutation.isPending && (
        <div className="flex items-center gap-2 py-2">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-500 border-t-cyan-400" />
          <span className="text-sm text-slate-400">Previewing changes...</span>
        </div>
      )}

      {previewMutation.isError && (
        <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          Failed to preview import. Check the file format and try again.
        </div>
      )}

      {preview && (
        <Card padding="sm">
          <p className="mb-2 text-sm font-medium text-slate-200">Preview (dry run)</p>
          {preview.errors.length > 0 && (
            <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {preview.errors.map((err, i) => (
                <p key={i}>{err}</p>
              ))}
            </div>
          )}
          {preview.changes.length > 0 ? (
            <div className="max-h-[30vh] overflow-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-white/10 text-left text-xs text-slate-400">
                    <th className="pb-2 pr-3">Action</th>
                    <th className="pb-2 pr-3">Item</th>
                    <th className="pb-2">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.changes.map((change, i) => (
                    <tr key={i} className="border-b border-white/5 text-slate-300">
                      <td className="py-1.5 pr-3 text-xs font-medium">{change.action}</td>
                      <td className="py-1.5 pr-3">{change.item}</td>
                      <td className="py-1.5 text-xs text-slate-400">
                        {change.details.join(", ")}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-slate-400">No changes detected.</p>
          )}
          <p className="mt-2 text-xs text-slate-400">{preview.summary}</p>

          {preview.changes.length > 0 && preview.errors.length === 0 && (
            <div className="mt-3 flex justify-end">
              <Button
                size="sm"
                onClick={() => applyMutation.mutate()}
                disabled={applyMutation.isPending}
              >
                {applyMutation.isPending ? "Applying..." : "Apply Changes"}
              </Button>
            </div>
          )}

          {applyMutation.isError && (
            <div className="mt-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              Failed to apply changes. Please try again.
            </div>
          )}

          {applyMutation.isSuccess && (
            <div className="mt-2 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-2 text-sm text-green-300">
              Changes applied successfully.
            </div>
          )}
        </Card>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// FeedbackHubModal
// ---------------------------------------------------------------------------

export function FeedbackHubModal({
  isOpen,
  onClose,
  feedbackSummary,
  activeKind,
  statusFilter,
  selectedNames,
  onDataChanged,
  initialTab = "review",
}: FeedbackHubModalProps) {
  const [activeTab, setActiveTab] = useState<string>(initialTab);

  useEffect(() => {
    if (isOpen) setActiveTab(initialTab);
  }, [isOpen, initialTab]);

  const items = feedbackSummary?.items ?? [];
  const pendingCount = items.length;

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Feedback Hub"
      maxWidth="max-w-4xl"
      testId={selectors.backlog.feedbackHub.modal}
    >
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="w-full">
          <TabsTrigger value="review" data-testid={selectors.backlog.feedbackHub.reviewTab}>
            Review
            {pendingCount > 0 && (
              <span className="ml-2 rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-300">
                {pendingCount}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="export" data-testid={selectors.backlog.feedbackHub.exportTab}>
            Export
          </TabsTrigger>
          <TabsTrigger value="import" data-testid={selectors.backlog.feedbackHub.importTab}>
            Import
          </TabsTrigger>
        </TabsList>

        <TabsContent value="review">
          <div
            className="mt-4 space-y-3"
            data-testid={selectors.backlog.feedbackHub.reviewTab}
          >
            {items.length === 0 ? (
              <p className="py-8 text-center text-sm text-slate-400">
                No pending feedback. All items are up to date.
              </p>
            ) : (
              items.map((item) => (
                <ReviewItemSection
                  key={`${item.kind}-${item.name}`}
                  item={item}
                  onSaved={onDataChanged}
                />
              ))
            )}
          </div>
        </TabsContent>

        <TabsContent value="export">
          <div className="mt-4">
            <ExportTab
              activeKind={activeKind}
              statusFilter={statusFilter}
              selectedNames={selectedNames}
            />
          </div>
        </TabsContent>

        <TabsContent value="import">
          <div className="mt-4">
            <ImportTab onDataChanged={onDataChanged} />
          </div>
        </TabsContent>
      </Tabs>
    </Dialog>
  );
}
