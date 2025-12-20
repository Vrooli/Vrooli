import { useState, useEffect } from "react";
import {
  GitCommit,
  FileText,
  Loader2,
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Plus,
  Minus,
  Edit3,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { useCommitPreview, useCommitPending, queryKeys } from "../lib/hooks";
import type { CommitPreviewFile, CommitPreviewSandboxGroup } from "../lib/api";
import { useQueryClient } from "@tanstack/react-query";

interface CommitPendingDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectRoot?: string;
}

// Group summary card
function SandboxGroupCard({
  group,
  expanded,
  onToggle,
}: {
  group: CommitPreviewSandboxGroup;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="w-full text-left rounded-lg border border-slate-700 bg-slate-800/50 p-3 hover:border-slate-600 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-slate-500" />
          ) : (
            <ChevronRight className="h-4 w-4 text-slate-500" />
          )}
          <span className="font-medium text-slate-200">{group.sandboxOwner || "Unknown"}</span>
          <Badge variant="default" className="text-[10px] bg-slate-700">
            {group.fileCount} files
          </Badge>
        </div>
        <div className="flex items-center gap-2 text-xs">
          {group.added > 0 && (
            <span className="text-emerald-400 flex items-center gap-0.5">
              <Plus className="h-3 w-3" />
              {group.added}
            </span>
          )}
          {group.modified > 0 && (
            <span className="text-amber-400 flex items-center gap-0.5">
              <Edit3 className="h-3 w-3" />
              {group.modified}
            </span>
          )}
          {group.deleted > 0 && (
            <span className="text-red-400 flex items-center gap-0.5">
              <Minus className="h-3 w-3" />
              {group.deleted}
            </span>
          )}
        </div>
      </div>
    </button>
  );
}

// File list item
function FileItem({ file }: { file: CommitPreviewFile }) {
  const typeColors = {
    added: "text-emerald-400",
    modified: "text-amber-400",
    deleted: "text-red-400",
  };
  const typeIcons = {
    added: Plus,
    modified: Edit3,
    deleted: Minus,
  };
  const Icon = typeIcons[file.changeType as keyof typeof typeIcons] || FileText;
  const color = typeColors[file.changeType as keyof typeof typeColors] || "text-slate-400";

  return (
    <div className="flex items-center gap-2 py-1 px-2 rounded text-xs hover:bg-slate-800/50">
      <Icon className={`h-3.5 w-3.5 ${color}`} />
      <span className="font-mono text-slate-300 flex-1 truncate">{file.relativePath}</span>
      {file.status === "already_committed" && (
        <Badge variant="default" className="text-[9px] bg-slate-700 text-slate-400">
          Already committed
        </Badge>
      )}
    </div>
  );
}

export function CommitPendingDialog({ open, onOpenChange, projectRoot }: CommitPendingDialogProps) {
  const queryClient = useQueryClient();
  const previewQuery = useCommitPreview(projectRoot);
  const commitMutation = useCommitPending();

  const [commitMessage, setCommitMessage] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [isEditing, setIsEditing] = useState(false);

  // Update commit message when preview loads
  useEffect(() => {
    if (previewQuery.data?.suggestedMessage && !isEditing) {
      setCommitMessage(previewQuery.data.suggestedMessage);
    }
  }, [previewQuery.data?.suggestedMessage, isEditing]);

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.commitPreview(projectRoot) });
  };

  const handleCommit = () => {
    commitMutation.mutate(
      {
        projectRoot,
        commitMessage,
      },
      {
        onSuccess: (result) => {
          if (result.success) {
            // Close the dialog on success
            onOpenChange(false);
          }
        },
      }
    );
  };

  const toggleGroup = (sandboxId: string) => {
    const newExpanded = new Set(expandedGroups);
    if (newExpanded.has(sandboxId)) {
      newExpanded.delete(sandboxId);
    } else {
      newExpanded.add(sandboxId);
    }
    setExpandedGroups(newExpanded);
  };

  const data = previewQuery.data;
  const committableFiles = data?.committableFiles ?? 0;
  const alreadyCommitted = data?.alreadyCommittedFiles ?? 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogClose onClose={() => onOpenChange(false)} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <GitCommit className="h-5 w-5 text-slate-400" />
            Commit Pending Changes
          </DialogTitle>
          <DialogDescription>
            Review and commit changes applied from sandboxes to the working tree.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 mt-4">
          {previewQuery.isLoading ? (
            <div className="flex items-center justify-center py-12 text-slate-500">
              <Loader2 className="h-5 w-5 animate-spin mr-2" />
              Loading pending changes...
            </div>
          ) : previewQuery.error ? (
            <div className="rounded-lg border border-red-800 bg-red-950/50 p-4 text-red-300 text-sm">
              <AlertCircle className="h-4 w-4 inline mr-2" />
              Failed to load pending changes: {(previewQuery.error as Error).message}
            </div>
          ) : !data || committableFiles === 0 ? (
            <div className="text-center py-12">
              <CheckCircle2 className="h-12 w-12 text-emerald-500 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-slate-200 mb-2">
                No uncommitted changes
              </h3>
              <p className="text-sm text-slate-500">
                All approved sandbox changes have been committed.
              </p>
              {alreadyCommitted > 0 && (
                <p className="text-xs text-slate-600 mt-2">
                  {alreadyCommitted} file(s) were already committed externally.
                </p>
              )}
            </div>
          ) : (
            <>
              {/* Summary */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 text-sm">
                  <span className="text-slate-400">
                    <span className="font-medium text-slate-200">{committableFiles}</span> files to commit
                  </span>
                  {alreadyCommitted > 0 && (
                    <span className="text-amber-400 flex items-center gap-1.5">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      {alreadyCommitted} already committed externally
                    </span>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleRefresh}
                  disabled={previewQuery.isFetching}
                  className="h-7 px-2"
                >
                  <RefreshCw className={`h-3.5 w-3.5 mr-1.5 ${previewQuery.isFetching ? "animate-spin" : ""}`} />
                  Refresh
                </Button>
              </div>

              {/* Sandbox groups */}
              <div className="space-y-2">
                <h4 className="text-xs uppercase tracking-wider text-slate-500 font-medium">
                  Changes by Sandbox
                </h4>
                {data.groupedBySandbox.map((group) => (
                  <div key={group.sandboxId}>
                    <SandboxGroupCard
                      group={group}
                      expanded={expandedGroups.has(group.sandboxId)}
                      onToggle={() => toggleGroup(group.sandboxId)}
                    />
                    {expandedGroups.has(group.sandboxId) && (
                      <div className="mt-1 ml-6 border-l border-slate-700 pl-4 space-y-0.5">
                        {data.files
                          .filter((f) => f.sandboxId === group.sandboxId && f.status === "pending")
                          .map((file) => (
                            <FileItem key={file.filePath} file={file} />
                          ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {/* Commit message editor */}
              <div className="space-y-2">
                <h4 className="text-xs uppercase tracking-wider text-slate-500 font-medium">
                  Commit Message
                </h4>
                <div className="relative">
                  <textarea
                    value={commitMessage}
                    onChange={(e) => {
                      setCommitMessage(e.target.value);
                      setIsEditing(true);
                    }}
                    onFocus={() => setIsEditing(true)}
                    rows={6}
                    className="w-full px-3 py-2 text-sm font-mono bg-slate-800 border border-slate-700 rounded-lg text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent resize-none"
                    placeholder="Enter commit message..."
                  />
                  {isEditing && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setCommitMessage(data.suggestedMessage);
                        setIsEditing(false);
                      }}
                      className="absolute top-2 right-2 h-6 text-xs text-slate-400"
                    >
                      Reset
                    </Button>
                  )}
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-4 border-t border-slate-800 mt-4">
          <div className="text-xs text-slate-500">
            {commitMutation.isSuccess && (
              <span className="text-emerald-400 flex items-center gap-1.5">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Committed {commitMutation.data?.filesCommitted} files
                {commitMutation.data?.commitHash && (
                  <span className="font-mono">({commitMutation.data.commitHash.slice(0, 7)})</span>
                )}
              </span>
            )}
            {commitMutation.error && (
              <span className="text-red-400 flex items-center gap-1.5">
                <AlertCircle className="h-3.5 w-3.5" />
                {(commitMutation.error as Error).message}
              </span>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCommit}
              disabled={committableFiles === 0 || commitMutation.isPending || !commitMessage.trim()}
              className="bg-emerald-600 hover:bg-emerald-500"
            >
              {commitMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Committing...
                </>
              ) : (
                <>
                  <GitCommit className="h-4 w-4 mr-2" />
                  Commit {committableFiles} files
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
