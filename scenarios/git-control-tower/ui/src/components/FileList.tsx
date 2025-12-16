import { useState } from "react";
import {
  File,
  FilePlus,
  FileX,
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  ChevronDown,
  ChevronRight,
  Loader2
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { RepoFilesStatus } from "../lib/api";

type FileCategory = "staged" | "unstaged" | "untracked" | "conflicts";

interface FileListProps {
  files?: RepoFilesStatus;
  selectedFile?: string;
  selectedIsStaged?: boolean;
  onSelectFile: (path: string, staged: boolean) => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  onDiscardFile: (path: string, untracked: boolean) => void;
  onStageAll: () => void;
  onUnstageAll: () => void;
  isStaging: boolean;
  isDiscarding: boolean;
  confirmingDiscard: string | null;
  onConfirmDiscard: (path: string | null) => void;
}

interface FileSectionProps {
  title: string;
  category: FileCategory;
  files: string[];
  icon: React.ReactNode;
  selectedFile?: string;
  selectedIsStaged?: boolean;
  onSelectFile: (path: string, staged: boolean) => void;
  onAction: (path: string) => void;
  actionIcon: React.ReactNode;
  actionLabel: string;
  isLoading: boolean;
  defaultExpanded?: boolean;
  onDiscard?: (path: string) => void;
  isDiscarding?: boolean;
  confirmingDiscard?: string | null;
  onConfirmDiscard?: (path: string | null) => void;
}

function FileSection({
  title,
  category,
  files,
  icon,
  selectedFile,
  selectedIsStaged,
  onSelectFile,
  onAction,
  actionIcon,
  actionLabel,
  isLoading,
  defaultExpanded = true,
  onDiscard,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard
}: FileSectionProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  if (files.length === 0) return null;

  const isStaged = category === "staged";
  const canDiscard = category === "unstaged" || category === "untracked";

  return (
    <div className="mb-4" data-testid={`file-section-${category}`}>
      <button
        className="flex items-center gap-2 w-full text-left px-2 py-1.5 hover:bg-slate-800/50 rounded transition-colors"
        onClick={() => setExpanded(!expanded)}
        data-testid={`file-section-toggle-${category}`}
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-slate-500" />
        ) : (
          <ChevronRight className="h-3 w-3 text-slate-500" />
        )}
        {icon}
        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        <span className="text-xs text-slate-600 ml-auto">{files.length}</span>
      </button>

      {expanded && (
        <ul className="mt-1 space-y-0.5">
          {files.map((file) => {
            const isSelected = selectedFile === file && selectedIsStaged === isStaged;
            const fileName = file.split("/").pop() || file;
            const dirPath = file.includes("/") ? file.substring(0, file.lastIndexOf("/")) : "";
            const isConfirmingThis = confirmingDiscard === file;

            return (
              <li
                key={file}
                className={`group flex items-center gap-2 px-2 py-1 rounded text-sm cursor-pointer transition-colors ${
                  isSelected
                    ? "bg-slate-700/50 text-slate-100"
                    : "hover:bg-slate-800/50 text-slate-300"
                }`}
                data-testid={`file-item-${category}`}
                data-file-path={file}
                onClick={() => onSelectFile(file, isStaged)}
              >
                <File className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                <div className="flex-1 min-w-0 overflow-hidden">
                  <span className="font-mono text-xs truncate block">
                    {dirPath && (
                      <span className="text-slate-500">{dirPath}/</span>
                    )}
                    {fileName}
                  </span>
                </div>

                {/* Confirmation UI for discard */}
                {isConfirmingThis && onConfirmDiscard && onDiscard && (
                  <div
                    className="flex items-center gap-1"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <span className="text-xs text-red-400 mr-1">Discard?</span>
                    <button
                      className="px-1.5 py-0.5 text-xs bg-red-600 hover:bg-red-500 text-white rounded transition-colors"
                      onClick={() => {
                        onDiscard(file);
                        onConfirmDiscard(null);
                      }}
                      disabled={isDiscarding}
                      data-testid="confirm-discard-yes"
                    >
                      Yes
                    </button>
                    <button
                      className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
                      onClick={() => onConfirmDiscard(null)}
                      data-testid="confirm-discard-no"
                    >
                      No
                    </button>
                  </div>
                )}

                {/* Action buttons (hidden during confirmation) */}
                {!isConfirmingThis && (
                  <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
                    <button
                      className="p-1 rounded hover:bg-slate-700 transition-all"
                      onClick={(e) => {
                        e.stopPropagation();
                        onAction(file);
                      }}
                      disabled={isLoading}
                      title={actionLabel}
                      data-testid={`file-action-${category}`}
                    >
                      {isLoading ? (
                        <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                      ) : (
                        actionIcon
                      )}
                    </button>
                    {canDiscard && onConfirmDiscard && (
                      <button
                        className="p-1 rounded hover:bg-red-900/50 transition-all"
                        onClick={(e) => {
                          e.stopPropagation();
                          onConfirmDiscard(file);
                        }}
                        disabled={isDiscarding}
                        title="Discard changes"
                        data-testid={`file-discard-${category}`}
                      >
                        {isDiscarding ? (
                          <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                        ) : (
                          <Trash2 className="h-3 w-3 text-red-400" />
                        )}
                      </button>
                    )}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

export function FileList({
  files,
  selectedFile,
  selectedIsStaged,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onStageAll,
  onUnstageAll,
  isStaging,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard
}: FileListProps) {
  const hasStaged = (files?.staged?.length ?? 0) > 0;
  const hasUnstaged = (files?.unstaged?.length ?? 0) > 0 || (files?.untracked?.length ?? 0) > 0;

  return (
    <Card className="h-full flex flex-col" data-testid="file-list-panel">
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <CardTitle>Changes</CardTitle>
        <div className="flex gap-2">
          {hasUnstaged && (
            <Button
              variant="outline"
              size="sm"
              onClick={onStageAll}
              disabled={isStaging}
              data-testid="stage-all-button"
            >
              <Plus className="h-3 w-3 mr-1" />
              Stage All
            </Button>
          )}
          {hasStaged && (
            <Button
              variant="outline"
              size="sm"
              onClick={onUnstageAll}
              disabled={isStaging}
              data-testid="unstage-all-button"
            >
              <Minus className="h-3 w-3 mr-1" />
              Unstage All
            </Button>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full px-2 py-2">
          {/* Conflicts - Always show first if any */}
          <FileSection
            title="Conflicts"
            category="conflicts"
            files={files?.conflicts ?? []}
            icon={<AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
            selectedFile={selectedFile}
            selectedIsStaged={selectedIsStaged}
            onSelectFile={onSelectFile}
            onAction={onStageFile}
            actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
            actionLabel="Stage file"
            isLoading={isStaging}
          />

          {/* Staged Changes */}
          <FileSection
            title="Staged"
            category="staged"
            files={files?.staged ?? []}
            icon={<FilePlus className="h-3.5 w-3.5 text-emerald-500" />}
            selectedFile={selectedFile}
            selectedIsStaged={selectedIsStaged}
            onSelectFile={onSelectFile}
            onAction={onUnstageFile}
            actionIcon={<Minus className="h-3 w-3 text-slate-400" />}
            actionLabel="Unstage file"
            isLoading={isStaging}
          />

          {/* Unstaged Changes */}
          <FileSection
            title="Modified"
            category="unstaged"
            files={files?.unstaged ?? []}
            icon={<FileX className="h-3.5 w-3.5 text-amber-500" />}
            selectedFile={selectedFile}
            selectedIsStaged={selectedIsStaged}
            onSelectFile={onSelectFile}
            onAction={onStageFile}
            actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
            actionLabel="Stage file"
            isLoading={isStaging}
            onDiscard={(path) => onDiscardFile(path, false)}
            isDiscarding={isDiscarding}
            confirmingDiscard={confirmingDiscard}
            onConfirmDiscard={onConfirmDiscard}
          />

          {/* Untracked Files */}
          <FileSection
            title="Untracked"
            category="untracked"
            files={files?.untracked ?? []}
            icon={<File className="h-3.5 w-3.5 text-slate-500" />}
            selectedFile={selectedFile}
            selectedIsStaged={selectedIsStaged}
            onSelectFile={onSelectFile}
            onAction={onStageFile}
            actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
            actionLabel="Stage file"
            isLoading={isStaging}
            defaultExpanded={false}
            onDiscard={(path) => onDiscardFile(path, true)}
            isDiscarding={isDiscarding}
            confirmingDiscard={confirmingDiscard}
            onConfirmDiscard={onConfirmDiscard}
          />

          {/* Empty State */}
          {files &&
           (files.staged?.length ?? 0) === 0 &&
           (files.unstaged?.length ?? 0) === 0 &&
           (files.untracked?.length ?? 0) === 0 &&
           (files.conflicts?.length ?? 0) === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="empty-state">
              <File className="h-8 w-8 text-slate-700 mb-3" />
              <p className="text-sm text-slate-500">No changes detected</p>
              <p className="text-xs text-slate-600 mt-1">
                Working directory is clean
              </p>
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
