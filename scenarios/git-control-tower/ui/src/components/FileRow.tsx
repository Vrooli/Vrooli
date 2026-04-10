import {
  memo,
  useContext,
} from "react";
import {
  File,
  Trash2,
  EyeOff,
  Binary,
  Loader2,
  ShieldCheck,
  MoreVertical,
  Square,
  CheckSquare,
  BarChart3,
} from "lucide-react";
import { useLongPress } from "../hooks";
import { resolveGroupForFile } from "../lib/grouping";
import { MobileContext, type FileRowProps } from "./FileListTypes";

export const FileRow = memo(function FileRow({
  file,
  displayPath,
  badge,
  isSelected,
  isStaged,
  canDiscard,
  isLoading,
  isDiscarding,
  isIgnoring,
  isBinary,
  isApproved,
  itemTestId,
  actionTestId,
  discardTestId,
  ignoreTestId,
  actionIcon,
  actionLabel,
  onSelectFile,
  onAction,
  onDiscard,
  onConfirmDiscard,
  onIgnore,
  onConfirmIgnore,
  confirmingDiscard,
  confirmingIgnore,
  groupingRules,
  onOpenMobileActions,
  onContextMenu,
  mobileSelectionMode,
  onLongPress,
  onMobileTap,
  onViewMetrics,
  category,
}: FileRowProps) {
  const isMobile = useContext(MobileContext);
  const isConfirmingIgnore = confirmingIgnore === file;
  const isConfirmingDiscard = confirmingDiscard === file;
  const showActionButtons = !(isConfirmingIgnore || isConfirmingDiscard) && !mobileSelectionMode;

  const longPressHandlers = useLongPress({
    onLongPress: () => {
      if (mobileSelectionMode) {
        onMobileTap?.(file, isStaged, "range");
      } else {
        onLongPress?.(file, isStaged);
      }
    },
    onTap: () => {
      if (mobileSelectionMode) {
        onMobileTap?.(file, isStaged, "toggle");
      }
      // When not in selection mode, tap is handled by onClick
    },
  });

  const touchProps = isMobile && onLongPress ? longPressHandlers : {};

  return (
    <li
      className={`group w-full flex items-center gap-2 rounded cursor-pointer transition-colors min-w-0 overflow-hidden select-none ${
        isMobile ? "px-2 py-1.5" : "px-2 py-1"
      } ${
        mobileSelectionMode && isSelected
          ? "bg-blue-900/30 ring-1 ring-blue-500/30 text-slate-100"
          : isSelected
            ? "bg-slate-700/50 text-slate-100"
            : "hover:bg-slate-800/50 active:bg-slate-700/50 text-slate-300"
      }`}
      data-testid={itemTestId}
      data-file-path={file}
      onClick={(event) => {
        // In mobile selection mode, taps are handled by useLongPress onTap
        if (isMobile && mobileSelectionMode) return;
        onSelectFile(file, isStaged, event);
      }}
      onContextMenu={onContextMenu ? (e) => {
        e.preventDefault();
        onContextMenu(file, e);
      } : undefined}
      {...touchProps}
    >
      {mobileSelectionMode && (
        <span className="flex-shrink-0 checkbox-appear">
          {isSelected ? (
            <CheckSquare className="h-5 w-5 text-blue-400" />
          ) : (
            <Square className="h-5 w-5 text-slate-500" />
          )}
        </span>
      )}
      {badge && (isMobile ? (
        <span
          className={`flex-shrink-0 rounded-full h-2.5 w-2.5 ${
            badge.label === "D" ? "bg-red-400" :
            badge.label === "M" ? "bg-amber-300" :
            badge.label === "A" ? "bg-emerald-300" :
            badge.label === "R" ? "bg-cyan-300" :
            badge.label === "U" ? "bg-red-300" :
            "bg-slate-400"
          }`}
          aria-label={`Status ${badge.label}`}
          title={`Status ${badge.label}`}
        />
      ) : (
        <span
          className={`flex items-center justify-center rounded border font-bold ${badge.style} h-5 w-5 text-[10px]`}
          aria-label={`Status ${badge.label}`}
          title={`Status ${badge.label}`}
        >
          {badge.label}
        </span>
      ))}
      {!isMobile && (
        <File
          className="text-slate-500 flex-shrink-0 h-3.5 w-3.5"
        />
      )}
      <div className="flex-1 min-w-0 overflow-hidden">
        <span className="font-mono text-xs truncate block w-full" title={file}>
          {displayPath}
        </span>
      </div>

      {isBinary && (
        <span
          className={`flex items-center gap-1 rounded border border-slate-700/60 bg-slate-900/60 px-1.5 py-0.5 text-slate-400 ${isMobile ? "text-xs" : "text-[10px]"}`}
          title="Binary file"
        >
          <Binary className="h-3 w-3" />
          bin
        </span>
      )}

      {isApproved && (
        <span
          className={`flex items-center gap-1 rounded border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0.5 text-emerald-300 ${isMobile ? "text-xs" : "text-[10px]"}`}
          title="Sandbox-approved change"
        >
          <ShieldCheck className="h-3 w-3" />
          approved
        </span>
      )}

      {isConfirmingIgnore && onConfirmIgnore && onIgnore && (() => {
        const group = groupingRules ? resolveGroupForFile(file, groupingRules) : null;
        return (
          <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
            {group ? (
              <>
                <span className="text-xs text-amber-300 mr-1">Ignore in:</span>
                <button
                  className="px-1.5 py-0.5 text-xs bg-blue-600 hover:bg-blue-500 text-white rounded transition-colors"
                  onClick={() => { onIgnore(file, "project"); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-project"
                >
                  Project
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-amber-500 hover:bg-amber-400 text-slate-900 rounded transition-colors"
                  onClick={() => { onIgnore(file, "group", group.groupDir); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-group"
                >
                  {group.groupLabel}
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
                  onClick={() => onConfirmIgnore(null)}
                  data-testid="confirm-ignore-cancel"
                >
                  Cancel
                </button>
              </>
            ) : (
              <>
                <span className="text-xs text-amber-300 mr-1">Ignore?</span>
                <button
                  className="px-1.5 py-0.5 text-xs bg-amber-500 hover:bg-amber-400 text-slate-900 rounded transition-colors"
                  onClick={() => { onIgnore(file); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-yes"
                >
                  Yes
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
                  onClick={() => onConfirmIgnore(null)}
                  data-testid="confirm-ignore-no"
                >
                  No
                </button>
              </>
            )}
          </div>
        );
      })()}

      {isConfirmingDiscard && onConfirmDiscard && onDiscard && (
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

      {showActionButtons && (
        <>
          {/* Desktop: hover-to-reveal actions */}
          {!isMobile && (
            <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
              <button
                className="p-1 rounded hover:bg-slate-700 transition-all"
                onClick={(e) => {
                  e.stopPropagation();
                  onAction(file);
                }}
                disabled={isLoading || isIgnoring}
                title={actionLabel}
                data-testid={actionTestId}
              >
                {isLoading ? (
                  <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                ) : (
                  actionIcon
                )}
              </button>
              {onViewMetrics && (
                <button
                  className="p-1 rounded hover:bg-slate-700 transition-all"
                  onClick={(e) => { e.stopPropagation(); if (category) onViewMetrics(file, category); }}
                  title="View file metrics"
                >
                  <BarChart3 className="h-3 w-3 text-slate-400" />
                </button>
              )}
              {onConfirmIgnore && onIgnore && (
                <button
                  className="p-1 rounded hover:bg-amber-900/40 transition-all"
                  onClick={(e) => {
                    e.stopPropagation();
                    onConfirmIgnore(file);
                  }}
                  disabled={isIgnoring}
                  title="Ignore file"
                  data-testid={ignoreTestId}
                >
                  {isIgnoring ? (
                    <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                  ) : (
                    <EyeOff className="h-3 w-3 text-amber-300" />
                  )}
                </button>
              )}
              {canDiscard && onConfirmDiscard && (
                <button
                  className="p-1 rounded hover:bg-red-900/50 transition-all"
                  onClick={(e) => {
                    e.stopPropagation();
                    onConfirmDiscard(file);
                  }}
                  disabled={isDiscarding || isIgnoring}
                  title="Discard changes"
                  data-testid={discardTestId}
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

          {/* Mobile: show primary action always, menu for secondary actions */}
          {isMobile && (
            <div className="flex items-center gap-1">
              {/* Primary action (stage/unstage) - always visible */}
              <button
                className="p-2 rounded-lg hover:bg-slate-700 active:bg-slate-600 transition-all touch-target"
                onClick={(e) => {
                  e.stopPropagation();
                  onAction(file);
                }}
                disabled={isLoading || isIgnoring}
                title={actionLabel}
                data-testid={actionTestId}
              >
                {isLoading ? (
                  <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
                ) : (
                  <span className="[&>svg]:h-5 [&>svg]:w-5">{actionIcon}</span>
                )}
              </button>

              {/* More actions button - opens bottom sheet */}
              {(onConfirmIgnore || canDiscard || onViewMetrics) && onOpenMobileActions && (
                <button
                  className="p-2 rounded-lg hover:bg-slate-700 active:bg-slate-600 transition-all touch-target"
                  onClick={(e) => {
                    e.stopPropagation();
                    onOpenMobileActions(file);
                  }}
                  title="More actions"
                  data-testid={`${itemTestId}-more-actions`}
                >
                  <MoreVertical className="h-5 w-5 text-slate-400" />
                </button>
              )}
            </div>
          )}
        </>
      )}
    </li>
  );
});

