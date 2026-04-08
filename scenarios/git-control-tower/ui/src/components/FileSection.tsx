import { useState, useMemo, useContext } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { formatPath } from "../lib/utils";
import { MobileContext, type FileSectionProps, LineStats, getStatusBadge } from "./FileListTypes";
import { FileRow } from "./FileRow";

export function FileSection({
  title,
  category,
  files,
  fileStatuses,
  binaryFiles,
  approvedFiles,
  maxPathChars,
  icon,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  onSelectFile,
  onAction,
  actionIcon,
  actionLabel,
  isLoading,
  changeStats,
  defaultExpanded = true,
  onDiscard,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard,
  onIgnore,
  isIgnoring,
  confirmingIgnore,
  onConfirmIgnore,
  groupingRules,
  onOpenMobileActions,
  onContextMenu,
  mobileSelectionMode,
  onLongPress,
  onMobileTap,
  onStatsClick,
  onViewMetrics,
}: FileSectionProps) {
  const isMobile = useContext(MobileContext);
  const [expanded, setExpanded] = useState(defaultExpanded);

  const isStaged = category === "staged";
  const canDiscard = category === "unstaged" || category === "untracked";
  const selectedKeys =
    selectedKeySet ?? new Set(selectedFiles?.map(selectionKey));

  const entries = useMemo(
    () =>
      files.map((file) => {
        const badge = getStatusBadge(fileStatuses?.[file], category);
        return {
          file,
          key: selectionKey({ path: file, staged: isStaged }),
          badge,
          displayPath: formatPath(file, maxPathChars),
          isBinary: binaryFiles?.has(file) ?? false,
          isApproved: approvedFiles?.has(file) ?? false,
        };
      }),
    [
      files,
      fileStatuses,
      category,
      maxPathChars,
      selectionKey,
      isStaged,
      binaryFiles,
      approvedFiles,
    ],
  );

  if (files.length === 0) return null;

  return (
    <div className="mb-4" data-testid={`file-section-${category}`}>
      <button
        className="flex items-center gap-2 w-full text-left px-2 py-1.5 hover:bg-slate-800/50 rounded transition-colors"
        onClick={() => setExpanded(!expanded)}
        data-testid={`file-section-toggle-${category}`}
      >
        {expanded ? (
          <ChevronDown className={`text-slate-500 ${isMobile ? "h-4 w-4" : "h-3 w-3"}`} />
        ) : (
          <ChevronRight className={`text-slate-500 ${isMobile ? "h-4 w-4" : "h-3 w-3"}`} />
        )}
        {icon}
        <span className={`font-medium text-slate-400 uppercase tracking-wider ${isMobile ? "text-sm" : "text-xs"}`}>
          {title}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <LineStats stats={changeStats} compact onClick={onStatsClick} />
          <span className={`text-slate-600 ${isMobile ? "text-sm" : "text-xs"}`}>{files.length}</span>
        </div>
      </button>

      {expanded && (
        <ul className="mt-1 space-y-0.5 min-w-0">
          {entries.map((entry) => (
            <FileRow
              key={entry.file}
              file={entry.file}
              displayPath={entry.displayPath}
              badge={entry.badge}
              isSelected={selectedKeys?.has(entry.key) ?? false}
              isStaged={isStaged}
              canDiscard={canDiscard}
              isLoading={isLoading}
              isDiscarding={isDiscarding ?? false}
              isIgnoring={isIgnoring ?? false}
              isBinary={entry.isBinary}
              isApproved={entry.isApproved}
              itemTestId={`file-item-${category}`}
              actionTestId={`file-action-${category}`}
              discardTestId={`file-discard-${category}`}
              ignoreTestId={`file-ignore-${category}`}
              actionIcon={actionIcon}
              actionLabel={actionLabel}
              onSelectFile={onSelectFile}
              onAction={onAction}
              onDiscard={onDiscard}
              onConfirmDiscard={onConfirmDiscard}
              onIgnore={onIgnore}
              onConfirmIgnore={onConfirmIgnore}
              confirmingDiscard={confirmingDiscard}
              confirmingIgnore={confirmingIgnore}
              groupingRules={groupingRules}
              onOpenMobileActions={onOpenMobileActions}
              onContextMenu={onContextMenu}
              mobileSelectionMode={mobileSelectionMode}
              onLongPress={onLongPress}
              onMobileTap={onMobileTap}
              onViewMetrics={onViewMetrics}
              category={category}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

