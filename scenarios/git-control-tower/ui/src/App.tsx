import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { FileList } from "./components/FileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import {
  useHealth,
  useRepoStatus,
  useDiff,
  useSyncStatus,
  useStageFiles,
  useUnstageFiles,
  useCommit,
  useDiscardFiles,
  usePush,
  usePull,
  queryKeys
} from "./lib/hooks";

export default function App() {
  const queryClient = useQueryClient();

  // Selected file state
  const [selectedFile, setSelectedFile] = useState<string | undefined>();
  const [selectedIsStaged, setSelectedIsStaged] = useState(false);
  const [confirmingDiscard, setConfirmingDiscard] = useState<string | null>(null);
  const [lastCommitHash, setLastCommitHash] = useState<string | undefined>();
  const [commitError, setCommitError] = useState<string | undefined>();

  // Queries
  const healthQuery = useHealth();
  const statusQuery = useRepoStatus();
  const syncStatusQuery = useSyncStatus();
  const diffQuery = useDiff(selectedFile, selectedIsStaged);

  // Mutations
  const stageMutation = useStageFiles();
  const unstageMutation = useUnstageFiles();
  const commitMutation = useCommit();
  const discardMutation = useDiscardFiles();
  const pushMutation = usePush();
  const pullMutation = usePull();

  const isStaging = stageMutation.isPending || unstageMutation.isPending;
  const isDiscarding = discardMutation.isPending;

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    if (selectedFile) {
      queryClient.invalidateQueries({
        queryKey: queryKeys.diff(selectedFile, selectedIsStaged)
      });
    }
  }, [queryClient, selectedFile, selectedIsStaged]);

  const handleSelectFile = useCallback((path: string, staged: boolean) => {
    setSelectedFile(path);
    setSelectedIsStaged(staged);
  }, []);

  const handleStageFile = useCallback(
    (path: string) => {
      stageMutation.mutate(
        { paths: [path] },
        {
          onSuccess: () => {
            // If we were viewing this file's unstaged diff, switch to staged
            if (selectedFile === path && !selectedIsStaged) {
              setSelectedIsStaged(true);
            }
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, false)
            });
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, true)
            });
          }
        }
      );
    },
    [stageMutation, queryClient, selectedFile, selectedIsStaged]
  );

  const handleUnstageFile = useCallback(
    (path: string) => {
      unstageMutation.mutate(
        { paths: [path] },
        {
          onSuccess: () => {
            // If we were viewing this file's staged diff, switch to unstaged
            if (selectedFile === path && selectedIsStaged) {
              setSelectedIsStaged(false);
            }
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, false)
            });
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, true)
            });
          }
        }
      );
    },
    [unstageMutation, queryClient, selectedFile, selectedIsStaged]
  );

  const handleStageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files) return;

    const allUnstaged = [
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
    if (allUnstaged.length === 0) return;

    stageMutation.mutate({ paths: allUnstaged });
  }, [stageMutation, statusQuery.data]);

  const handleUnstageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files || (files.staged?.length ?? 0) === 0) return;

    unstageMutation.mutate({ paths: files.staged ?? [] });
  }, [unstageMutation, statusQuery.data]);

  const handleDiscardFile = useCallback(
    (path: string, untracked: boolean) => {
      discardMutation.mutate(
        { paths: [path], untracked },
        {
          onSuccess: () => {
            // If we were viewing this file's diff, clear selection
            if (selectedFile === path) {
              setSelectedFile(undefined);
            }
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
          }
        }
      );
    },
    [discardMutation, queryClient, selectedFile]
  );

  const handleCommit = useCallback(
    (message: string, conventional: boolean) => {
      setCommitError(undefined);
      setLastCommitHash(undefined);

      commitMutation.mutate(
        { message, conventional },
        {
          onSuccess: (result) => {
            if (result.success && result.hash) {
              setLastCommitHash(result.hash);
              // Clear selection if viewing staged diff
              if (selectedIsStaged) {
                setSelectedFile(undefined);
              }
            } else {
              setCommitError(
                result.error ||
                  result.validation_errors?.join("; ") ||
                  "Commit failed"
              );
            }
          },
          onError: (error) => {
            setCommitError(error.message);
          }
        }
      );
    },
    [commitMutation, selectedIsStaged]
  );

  const handlePush = useCallback(() => {
    pushMutation.mutate({});
  }, [pushMutation]);

  const handlePull = useCallback(() => {
    pullMutation.mutate({});
  }, [pullMutation]);

  return (
    <div
      className="h-screen flex flex-col bg-slate-950 text-slate-50"
      data-testid="git-control-tower"
    >
      {/* Status Header */}
      <StatusHeader
        status={statusQuery.data}
        health={healthQuery.data}
        syncStatus={syncStatusQuery.data}
        isLoading={statusQuery.isLoading || healthQuery.isLoading}
        onRefresh={handleRefresh}
        onPush={handlePush}
        onPull={handlePull}
        isPushing={pushMutation.isPending}
        isPulling={pullMutation.isPending}
      />

      {/* Main Content - Split Pane */}
      <div className="flex-1 flex overflow-hidden">
        {/* File List Panel + Commit Panel */}
        <div className="w-80 flex-shrink-0 border-r border-slate-800 overflow-hidden flex flex-col">
          <div className="flex-1 overflow-hidden">
            <FileList
              files={statusQuery.data?.files}
              selectedFile={selectedFile}
              selectedIsStaged={selectedIsStaged}
              onSelectFile={handleSelectFile}
              onStageFile={handleStageFile}
              onUnstageFile={handleUnstageFile}
              onDiscardFile={handleDiscardFile}
              onStageAll={handleStageAll}
              onUnstageAll={handleUnstageAll}
              isStaging={isStaging}
              isDiscarding={isDiscarding}
              confirmingDiscard={confirmingDiscard}
              onConfirmDiscard={setConfirmingDiscard}
            />
          </div>
          <div className="flex-shrink-0 border-t border-slate-800">
            <CommitPanel
              stagedCount={statusQuery.data?.summary.staged ?? 0}
              onCommit={handleCommit}
              isCommitting={commitMutation.isPending}
              lastCommitHash={lastCommitHash}
              commitError={commitError}
            />
          </div>
        </div>

        {/* Diff Viewer Panel */}
        <div className="flex-1 overflow-hidden">
          <DiffViewer
            diff={diffQuery.data}
            selectedFile={selectedFile}
            isStaged={selectedIsStaged}
            isLoading={diffQuery.isLoading}
            error={diffQuery.error}
          />
        </div>
      </div>

      {/* Error Toast for Mutations */}
      {(stageMutation.error ||
        unstageMutation.error ||
        discardMutation.error ||
        pushMutation.error ||
        pullMutation.error) && (
        <div
          className="fixed bottom-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md"
          data-testid="error-toast"
        >
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">
            {(
              stageMutation.error ||
              unstageMutation.error ||
              discardMutation.error ||
              pushMutation.error ||
              pullMutation.error
            )?.message}
          </p>
        </div>
      )}
    </div>
  );
}
