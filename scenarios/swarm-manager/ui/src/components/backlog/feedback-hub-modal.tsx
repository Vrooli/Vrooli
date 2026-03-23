import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Copy,
  Check,
  Code,
  Download,
  Eye,
  Filter,
  RefreshCw,
  Upload,
  ChevronDown,
  ChevronRight,
  Info,
} from "lucide-react";
import { renderMarkdown } from "../../lib/render-markdown";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../ui/tabs";
import { selectors } from "../../consts/selectors";
import { backlogService } from "../../services";
import type { ImportBacklogResponse } from "../../services/backlog-service";
import {
  BACKLOG_KINDS,
  BACKLOG_KIND_LABELS,
  BACKLOG_STATUSES,
  formatBacklogStatus,
} from "../../types";
import type {
  BacklogKind,
  BacklogStatus,
  FeedbackSummaryResponse,
  FeedbackItemSummary,
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

// ---------------------------------------------------------------------------
// Review Tab - Expandable item sections showing pending workshop proposals
// ---------------------------------------------------------------------------

function ReviewItemSection({
  item,
}: {
  item: FeedbackItemSummary;
  onSaved: () => void;
}) {
  const [expanded, setExpanded] = useState(true);

  const counts: string[] = [];
  if (item.pending_decisions > 0)
    counts.push(`${item.pending_decisions} decision${item.pending_decisions === 1 ? "" : "s"}`);

  const totalPending = item.pending_decisions;

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
          {totalPending > 0 ? (
            <div className="flex items-start gap-3 rounded-lg border border-cyan-500/20 bg-cyan-500/5 px-4 py-3">
              <Info className="mt-0.5 h-4 w-4 shrink-0 text-cyan-400" />
              <div className="text-sm text-slate-300">
                <p>
                  This item has {totalPending} pending workshop item{totalPending === 1 ? "" : "s"} that need{totalPending === 1 ? "s" : ""} your response.
                </p>
                <p className="mt-1 text-xs text-slate-400">
                  Visit the item detail page to review and respond to workshop proposals.
                </p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400">No pending items for this entry.</p>
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
  isActive,
}: {
  activeKind: TabKind;
  statusFilter: BacklogStatus | "";
  selectedNames?: string[];
  isActive: boolean;
}) {
  // Content toggles
  const [includePrd, setIncludePrd] = useState(false);
  const [includeRequirements, setIncludeRequirements] = useState(false);
  const [includeClarifyQuestions, setIncludeClarifyQuestions] = useState(true);
  const [includeSuggestions, setIncludeSuggestions] = useState(true);
  const [includeNotes, setIncludeNotes] = useState(true);
  const [includeTemplate, setIncludeTemplate] = useState(true);

  // Filter state
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filterKinds, setFilterKinds] = useState<Set<BacklogKind>>(() =>
    activeKind !== "all" ? new Set([activeKind]) : new Set(BACKLOG_KINDS),
  );
  const [filterStatuses, setFilterStatuses] = useState<Set<BacklogStatus>>(() =>
    statusFilter ? new Set([statusFilter]) : new Set(BACKLOG_STATUSES.filter((s) => s !== "archived")),
  );
  const [filterPriorityMax, setFilterPriorityMax] = useState<number | "">("");
  const [filterTags, setFilterTags] = useState<Set<string>>(new Set());

  // View state
  const [content, setContent] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");
  const hasAutoGenerated = useRef(false);

  // Fetch available tags from backlog items (only when tab is active).
  const tagsQuery = useQuery({
    queryKey: ["backlog", "export-tags"],
    queryFn: async () => {
      const items = await backlogService.list();
      const tagSet = new Set<string>();
      for (const item of items) {
        for (const tag of item.tags ?? []) {
          tagSet.add(tag);
        }
      }
      return Array.from(tagSet).sort();
    },
    enabled: isActive,
    staleTime: 60_000,
  });

  // Count active filters (non-default).
  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (filterKinds.size !== BACKLOG_KINDS.length) count++;
    const defaultStatuses = BACKLOG_STATUSES.filter((s) => s !== "archived");
    if (filterStatuses.size !== defaultStatuses.length || defaultStatuses.some((s) => !filterStatuses.has(s))) count++;
    if (filterPriorityMax !== "") count++;
    if (filterTags.size > 0) count++;
    return count;
  }, [filterKinds, filterStatuses, filterPriorityMax, filterTags]);

  const exportMutation = useMutation({
    mutationFn: async () => {
      const params: Parameters<typeof backlogService.exportItems>[0] = {
        includePrd,
        includeRequirements,
        includeClarifyQuestions,
        includeSuggestions,
        includeNotes,
        includeTemplate,
      };
      if (filterKinds.size > 0 && filterKinds.size < BACKLOG_KINDS.length) {
        params.kinds = Array.from(filterKinds);
      }
      if (filterStatuses.size > 0) {
        params.statuses = Array.from(filterStatuses);
      }
      if (selectedNames && selectedNames.length > 0) params.names = selectedNames;
      if (filterPriorityMax !== "") params.priorityMax = filterPriorityMax;
      if (filterTags.size > 0) params.tags = Array.from(filterTags);
      const blob = await backlogService.exportItems(params);
      return blob.text();
    },
    onSuccess: (text) => setContent(text),
  });

  // Auto-generate on tab activation.
  useEffect(() => {
    if (isActive && !hasAutoGenerated.current && content === null && !exportMutation.isPending) {
      hasAutoGenerated.current = true;
      exportMutation.mutate();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isActive]);

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

  const handleRefresh = () => {
    setContent(null);
    setCopied(false);
    exportMutation.mutate();
  };

  const toggleKind = (kind: BacklogKind) => {
    setFilterKinds((prev) => {
      const next = new Set(prev);
      if (next.has(kind)) next.delete(kind);
      else next.add(kind);
      return next;
    });
  };

  const toggleStatus = (status: BacklogStatus) => {
    setFilterStatuses((prev) => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  };

  const toggleTag = (tag: string) => {
    setFilterTags((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });
  };

  const checkboxClass = "rounded border-white/20 bg-slate-800";

  return (
    <div className="space-y-4" data-testid={selectors.backlog.feedbackHub.exportTab}>
      {/* Info banner */}
      <div className="flex items-start gap-3 rounded-lg border border-cyan-500/20 bg-cyan-500/5 px-4 py-3">
        <Info className="mt-0.5 h-4 w-4 shrink-0 text-cyan-400" />
        <div className="text-sm text-slate-300">
          <p>Export your backlog as markdown for sharing, review, or editing outside the UI.</p>
          <p className="mt-1 text-xs text-slate-400">
            Exported files can be re-imported to apply changes. Use options below to control content.
          </p>
        </div>
      </div>

      {/* Collapsible filters */}
      <div className="rounded-lg border border-white/10 bg-slate-800/40">
        <button
          type="button"
          onClick={() => setFiltersOpen((prev) => !prev)}
          className="flex w-full items-center gap-2 px-4 py-3 text-left text-sm font-medium text-slate-100 hover:bg-slate-800/60 transition-colors"
        >
          <Filter className="h-4 w-4 text-slate-400" />
          <span>Filters</span>
          {activeFilterCount > 0 && (
            <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-300">
              {activeFilterCount}
            </span>
          )}
          <span className="ml-auto">
            {filtersOpen ? (
              <ChevronDown className="h-4 w-4 text-slate-400" />
            ) : (
              <ChevronRight className="h-4 w-4 text-slate-400" />
            )}
          </span>
        </button>

        {filtersOpen && (
          <div className="border-t border-white/5 px-4 py-4 space-y-4">
            {/* Kind filter */}
            <div>
              <p className="mb-2 text-xs font-medium text-slate-400 uppercase tracking-wider">Kind</p>
              <div className="flex flex-wrap gap-3">
                {BACKLOG_KINDS.map((kind) => (
                  <label key={kind} className="flex items-center gap-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      checked={filterKinds.has(kind)}
                      onChange={() => toggleKind(kind)}
                      className={checkboxClass}
                    />
                    {BACKLOG_KIND_LABELS[kind]}
                  </label>
                ))}
              </div>
            </div>

            {/* Status filter */}
            <div>
              <p className="mb-2 text-xs font-medium text-slate-400 uppercase tracking-wider">Status</p>
              <div className="flex flex-wrap gap-3">
                {BACKLOG_STATUSES.map((status) => (
                  <label key={status} className="flex items-center gap-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      checked={filterStatuses.has(status)}
                      onChange={() => toggleStatus(status)}
                      className={checkboxClass}
                    />
                    {formatBacklogStatus(status)}
                  </label>
                ))}
              </div>
            </div>

            {/* Priority max */}
            <div>
              <p className="mb-2 text-xs font-medium text-slate-400 uppercase tracking-wider">Priority max</p>
              <input
                type="number"
                min={1}
                max={10}
                value={filterPriorityMax}
                onChange={(e) => {
                  const val = e.target.value;
                  setFilterPriorityMax(val === "" ? "" : Math.min(10, Math.max(1, Number(val))));
                }}
                placeholder="No limit"
                className="w-28 rounded-md border border-white/10 bg-slate-800 px-3 py-1.5 text-sm text-slate-200 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none"
              />
            </div>

            {/* Tags filter */}
            {tagsQuery.data && tagsQuery.data.length > 0 && (
              <div>
                <p className="mb-2 text-xs font-medium text-slate-400 uppercase tracking-wider">Tags</p>
                <div className="flex flex-wrap gap-3">
                  {tagsQuery.data.map((tag) => (
                    <label key={tag} className="flex items-center gap-2 text-sm text-slate-200">
                      <input
                        type="checkbox"
                        checked={filterTags.has(tag)}
                        onChange={() => toggleTag(tag)}
                        className={checkboxClass}
                      />
                      {tag}
                    </label>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Content toggles */}
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includePrd} onChange={(e) => setIncludePrd(e.target.checked)} className={checkboxClass} />
          Include PRD
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includeRequirements} onChange={(e) => setIncludeRequirements(e.target.checked)} className={checkboxClass} />
          Include Requirements
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includeClarifyQuestions} onChange={(e) => setIncludeClarifyQuestions(e.target.checked)} className={checkboxClass} />
          Clarify Questions
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includeSuggestions} onChange={(e) => setIncludeSuggestions(e.target.checked)} className={checkboxClass} />
          Suggestions
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includeNotes} onChange={(e) => setIncludeNotes(e.target.checked)} className={checkboxClass} />
          Notes
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input type="checkbox" checked={includeTemplate} onChange={(e) => setIncludeTemplate(e.target.checked)} className={checkboxClass} />
          New Item Template
        </label>
      </div>

      {/* Refresh button */}
      <Button
        size="sm"
        onClick={handleRefresh}
        disabled={exportMutation.isPending}
      >
        {exportMutation.isPending ? (
          <>
            <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
            Generating...
          </>
        ) : (
          <>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh Export
          </>
        )}
      </Button>

      {exportMutation.isError && (
        <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {exportMutation.error instanceof Error ? exportMutation.error.message : "Failed to generate export. Please try again."}
        </div>
      )}

      {content !== null && (
        <>
          {/* View mode toggle + preview */}
          <div className="rounded-lg border border-slate-700/70 bg-slate-800/60 overflow-hidden">
            <div className="flex items-center justify-between border-b border-white/10 px-3 py-2">
              <span className="text-xs text-slate-400">
                {viewMode === "rendered" ? "Rendered preview" : "Raw markdown"}
              </span>
              <button
                type="button"
                onClick={() => setViewMode((prev) => (prev === "rendered" ? "raw" : "rendered"))}
                className="inline-flex h-8 w-8 items-center justify-center rounded-full text-slate-300 transition-colors hover:bg-slate-700/60 hover:text-white"
                aria-label={viewMode === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
                title={viewMode === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
              >
                {viewMode === "rendered" ? <Code className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            <div className="max-h-[40vh] overflow-auto p-4">
              {viewMode === "rendered" ? (
                <div
                  className="prose prose-invert max-w-none prose-headings:mb-2 prose-headings:mt-4 prose-p:my-2 prose-pre:bg-slate-900 prose-code:text-cyan-300"
                  dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
                />
              ) : (
                <pre className="text-sm text-slate-300 font-mono whitespace-pre-wrap break-words">
                  {content}
                </pre>
              )}
            </div>
          </div>

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
          {previewMutation.error instanceof Error ? previewMutation.error.message : "Failed to preview import. Check the file format and try again."}
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
              isActive={activeTab === "export"}
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
