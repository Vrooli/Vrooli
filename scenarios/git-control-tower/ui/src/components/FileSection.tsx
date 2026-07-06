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
  pendingPaths,
  isLoading,
  changeStats,
  defaultExpanded = true,
  expanded: controlledExpanded,
  onToggle,
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
  // Controlled when `expanded`/`onToggle` are supplied (persisted by the parent);
  // otherwise fall back to local uncontrolled state.
  const [internalExpanded, setInternalExpanded] = useState(defaultExpanded);
  const isControlled = controlledExpanded !== undefined;
  const expanded = isControlled ? controlledExpanded : internalExpanded;
  const handleToggle = () => {
    if (isControlled) {
      onToggle?.();
    } else {
      setInternalExpanded((prev) => !prev);
    }
  };

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
      <div className="flex items-center gap-2 w-full px-2 py-1.5 rounded hover:bg-slate-800/50 transition-colors">
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
          onClick={handleToggle}
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
        </button>
        <div className="ml-auto flex items-center gap-2">
          <LineStats stats={changeStats} compact onClick={onStatsClick} />
          <span className={`text-slate-600 ${isMobile ? "text-sm" : "text-xs"}`}>{files.length}</span>
        </div>
      </div>

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
              isLoading={pendingPaths?.has(entry.file) ?? isLoading ?? false}
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
