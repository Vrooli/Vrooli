import {
  Plus,
  Minus,
  Trash2,
  EyeOff,
  BarChart3,
} from "lucide-react";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import type { ChangeGroupAPI } from "../lib/api";
import type { FileCategory } from "./FileListTypes";

interface MobileActionFileInfo {
  path: string;
  isStaged: boolean;
  isUnstaged: boolean;
  isUntracked: boolean;
  isConflict: boolean;
}

export interface FileListMobileActionsProps {
  mobileActionFile: string | null;
  mobileActionFileInfo: MobileActionFileInfo | null;
  onClose: () => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  onDiscardFile: (path: string, untracked: boolean) => void;
  onIgnoreFile: (path: string, level?: "project" | "group", groupDir?: string) => void;
  openFileMetrics: (path: string, category?: FileCategory) => void;
  resolvedGroups?: ChangeGroupAPI[];
}

export function FileListMobileActions({
  mobileActionFile,
  mobileActionFileInfo,
  onClose,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onIgnoreFile,
  openFileMetrics,
  resolvedGroups,
}: FileListMobileActionsProps) {
  if (!mobileActionFileInfo) return null;

  return (
    <BottomSheet
      isOpen={Boolean(mobileActionFile)}
      onClose={onClose}
      title={
        mobileActionFileInfo.path.split("/").pop() ||
        mobileActionFileInfo.path
      }
    >
      <div className="space-y-1">
        {/* View metrics */}
        <BottomSheetAction
          icon={<BarChart3 className="h-5 w-5 text-slate-300" />}
          label="View Metrics"
          description="View change metrics for this file"
          onClick={() => {
            const cat: FileCategory = mobileActionFileInfo.isStaged ? "staged"
              : mobileActionFileInfo.isUntracked ? "untracked"
              : "unstaged";
            openFileMetrics(mobileActionFileInfo.path, cat);
            onClose();
          }}
        />

        {/* Stage/Unstage action */}
        {mobileActionFileInfo.isStaged && (
          <BottomSheetAction
            icon={<Minus className="h-5 w-5 text-slate-300" />}
            label="Unstage"
            description="Remove from staged changes"
            onClick={() => {
              onUnstageFile(mobileActionFileInfo.path);
              onClose();
            }}
          />
        )}
        {(mobileActionFileInfo.isUnstaged ||
          mobileActionFileInfo.isUntracked ||
          mobileActionFileInfo.isConflict) && (
          <BottomSheetAction
            icon={<Plus className="h-5 w-5 text-emerald-300" />}
            label="Stage"
            description="Add to staged changes"
            onClick={() => {
              onStageFile(mobileActionFileInfo.path);
              onClose();
            }}
          />
        )}

        {/* Ignore action */}
        {(() => {
          const group = resolvedGroups?.find((candidate) =>
            candidate.source !== "builtin" && candidate.files.includes(mobileActionFileInfo.path),
          );
          if (group) {
            return (
              <>
                <BottomSheetAction
                  icon={<EyeOff className="h-5 w-5 text-blue-400" />}
                  label="Ignore (Project)"
                  description="Add to root .gitignore"
                  onClick={() => {
                    onIgnoreFile(mobileActionFileInfo.path, "project");
                    onClose();
                  }}
                />
                <BottomSheetAction
                  icon={<EyeOff className="h-5 w-5 text-amber-300" />}
                  label={`Ignore (${group.label})`}
                  description={`Add to ${group.root ?? ""}.gitignore`}
                  onClick={() => {
                    onIgnoreFile(mobileActionFileInfo.path, "group", group.root ?? "");
                    onClose();
                  }}
                />
              </>
            );
          }
          return (
            <BottomSheetAction
              icon={<EyeOff className="h-5 w-5 text-amber-300" />}
              label="Ignore"
              description="Add to .gitignore"
              onClick={() => {
                onIgnoreFile(mobileActionFileInfo.path);
                onClose();
              }}
            />
          );
        })()}

        {/* Discard action - only for unstaged/untracked */}
        {(mobileActionFileInfo.isUnstaged ||
          mobileActionFileInfo.isUntracked) && (
          <BottomSheetAction
            icon={<Trash2 className="h-5 w-5 text-red-400" />}
            label="Discard Changes"
            description={
              mobileActionFileInfo.isUntracked
                ? "Delete this file"
                : "Revert to last commit"
            }
            variant="danger"
            onClick={() => {
              onDiscardFile(
                mobileActionFileInfo.path,
                mobileActionFileInfo.isUntracked,
              );
              onClose();
            }}
          />
        )}
      </div>
    </BottomSheet>
  );
}
