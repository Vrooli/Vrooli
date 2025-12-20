import { useState, useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { SandboxList } from "./components/SandboxList";
import { SandboxDetail } from "./components/SandboxDetail";
import { CreateSandboxDialog } from "./components/CreateSandboxDialog";
import { SettingsDialog } from "./components/SettingsDialog";
import { CommitPendingDialog } from "./components/CommitPendingDialog";
import { LaunchAgentDialog, type LaunchConfig } from "./components/LaunchAgentDialog";
import {
  useHealth,
  useSandboxes,
  useDiff,
  useCreateSandbox,
  useDeleteSandbox,
  useStopSandbox,
  useStartSandbox,
  useApproveSandbox,
  useRejectSandbox,
  useDiscardFiles,
  useExecCommand,
  useStartProcess,
  queryKeys,
} from "./lib/hooks";
import { computeStats, type Sandbox, type CreateRequest } from "./lib/api";
import { SELECTORS } from "./consts/selectors";

export default function App() {
  const queryClient = useQueryClient();

  // Local state
  const [selectedSandbox, setSelectedSandbox] = useState<Sandbox | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [commitDialogOpen, setCommitDialogOpen] = useState(false);
  const [launchDialogOpen, setLaunchDialogOpen] = useState(false);

  // Queries
  const healthQuery = useHealth();
  const sandboxesQuery = useSandboxes();
  const diffQuery = useDiff(selectedSandbox?.id);

  // Mutations
  const createMutation = useCreateSandbox();
  const deleteMutation = useDeleteSandbox();
  const stopMutation = useStopSandbox();
  const startMutation = useStartSandbox();
  const approveMutation = useApproveSandbox();
  const rejectMutation = useRejectSandbox();
  const discardMutation = useDiscardFiles();
  const execMutation = useExecCommand();
  const startProcessMutation = useStartProcess();

  // Computed stats
  const stats = useMemo(() => {
    if (!sandboxesQuery.data?.sandboxes) return undefined;
    return computeStats(sandboxesQuery.data.sandboxes);
  }, [sandboxesQuery.data?.sandboxes]);

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    if (selectedSandbox?.id) {
      queryClient.invalidateQueries({ queryKey: queryKeys.diff(selectedSandbox.id) });
    }
  }, [queryClient, selectedSandbox?.id]);

  const handleSelectSandbox = useCallback((sandbox: Sandbox) => {
    setSelectedSandbox(sandbox);
  }, []);

  const handleCreate = useCallback(
    (req: CreateRequest) => {
      createMutation.mutate(req, {
        onSuccess: (newSandbox) => {
          setCreateDialogOpen(false);
          setSelectedSandbox(newSandbox);
        },
      });
    },
    [createMutation]
  );

  const handleStop = useCallback(() => {
    if (!selectedSandbox) return;
    stopMutation.mutate(selectedSandbox.id, {
      onSuccess: (updated) => {
        setSelectedSandbox(updated);
      },
    });
  }, [selectedSandbox, stopMutation]);

  const handleStart = useCallback(() => {
    if (!selectedSandbox) return;
    startMutation.mutate(selectedSandbox.id, {
      onSuccess: (updated) => {
        setSelectedSandbox(updated);
      },
    });
  }, [selectedSandbox, startMutation]);

  const handleApprove = useCallback(() => {
    if (!selectedSandbox) return;
    approveMutation.mutate(
      { id: selectedSandbox.id },
      {
        onSuccess: () => {
          // Refresh the sandbox to get updated status
          queryClient.invalidateQueries({
            queryKey: queryKeys.sandbox(selectedSandbox.id),
          });
          queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
        },
      }
    );
  }, [selectedSandbox, approveMutation, queryClient]);

  const handleApproveAll = useCallback(() => {
    if (!selectedSandbox) return;
    approveMutation.mutate(
      { id: selectedSandbox.id, options: { approveAll: true } },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: queryKeys.sandbox(selectedSandbox.id),
          });
          queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
        },
      }
    );
  }, [selectedSandbox, approveMutation, queryClient]);

  const handleReject = useCallback(() => {
    if (!selectedSandbox) return;
    rejectMutation.mutate(
      { id: selectedSandbox.id },
      {
        onSuccess: (updated) => {
          setSelectedSandbox(updated);
        },
      }
    );
  }, [selectedSandbox, rejectMutation]);

  const handleDelete = useCallback(() => {
    if (!selectedSandbox) return;
    deleteMutation.mutate(selectedSandbox.id, {
      onSuccess: () => {
        setSelectedSandbox(null);
      },
    });
  }, [selectedSandbox, deleteMutation]);

  const handleDiscardFile = useCallback(
    (fileId: string) => {
      if (!selectedSandbox) return;
      discardMutation.mutate({
        sandboxId: selectedSandbox.id,
        fileIds: [fileId],
      });
    },
    [selectedSandbox, discardMutation]
  );

  const handleLaunch = useCallback(
    (config: LaunchConfig) => {
      if (!selectedSandbox) return;

      const request = {
        command: config.command,
        args: config.args.length > 0 ? config.args : undefined,
        isolationLevel: config.isolationProfile, // Backend expects isolationLevel but accepts profile IDs
        memoryLimitMB: config.memoryLimitMB,
        cpuTimeSec: config.cpuTimeSec,
        maxProcesses: config.maxProcesses,
        maxOpenFiles: config.maxOpenFiles,
        allowNetwork: config.allowNetwork,
        env: Object.keys(config.env).length > 0 ? config.env : undefined,
        workingDir: config.workingDir !== "/workspace" ? config.workingDir : undefined,
      };

      if (config.mode === "run") {
        startProcessMutation.mutate(
          {
            sandboxId: selectedSandbox.id,
            request: {
              ...request,
              name: config.name,
            },
          },
          {
            onSuccess: () => {
              setLaunchDialogOpen(false);
            },
          }
        );
      } else {
        execMutation.mutate(
          {
            sandboxId: selectedSandbox.id,
            request: {
              ...request,
              timeoutSec: config.timeoutSec,
            },
          },
          {
            onSuccess: () => {
              setLaunchDialogOpen(false);
            },
          }
        );
      }
    },
    [selectedSandbox, execMutation, startProcessMutation]
  );

  // Keep selected sandbox in sync with list updates
  const sandboxes = sandboxesQuery.data?.sandboxes || [];

  // Update selected sandbox from list if it was updated
  const selectedFromList = selectedSandbox
    ? sandboxes.find((sb) => sb.id === selectedSandbox.id)
    : null;

  // Use the list version if available (more up-to-date)
  const currentSandbox = selectedFromList || selectedSandbox;

  // Extract existing reserved paths from active sandboxes for conflict detection
  const existingReservedPaths = useMemo(() => {
    return sandboxes
      .filter((sb) => sb.status === "active" || sb.status === "creating")
      .map((sb) => sb.reservedPath || sb.scopePath);
  }, [sandboxes]);

  return (
    <div
      className="h-screen flex flex-col bg-slate-950 text-slate-50"
      data-testid={SELECTORS.app}
    >
      {/* Status Header */}
      <StatusHeader
        health={healthQuery.data}
        stats={stats}
        isLoading={healthQuery.isLoading || sandboxesQuery.isLoading}
        onRefresh={handleRefresh}
        onCreateClick={() => setCreateDialogOpen(true)}
        onSettingsClick={() => setSettingsDialogOpen(true)}
        onCommitClick={() => setCommitDialogOpen(true)}
      />

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sandbox List - Left Panel */}
        <div className="w-80 flex-shrink-0 border-r border-slate-800 overflow-hidden">
          <SandboxList
            sandboxes={sandboxes}
            selectedId={currentSandbox?.id}
            onSelect={handleSelectSandbox}
            isLoading={sandboxesQuery.isLoading}
          />
        </div>

        {/* Detail Panel - Right Panel */}
        <div className="flex-1 overflow-hidden">
          <SandboxDetail
            sandbox={currentSandbox || undefined}
            diff={diffQuery.data}
            isDiffLoading={diffQuery.isLoading}
            diffError={diffQuery.error}
            onStop={handleStop}
            onStart={handleStart}
            onApprove={handleApprove}
            onApproveAll={handleApproveAll}
            onReject={handleReject}
            onDelete={handleDelete}
            onDiscardFile={handleDiscardFile}
            onLaunchAgent={() => setLaunchDialogOpen(true)}
            isApproving={approveMutation.isPending}
            isRejecting={rejectMutation.isPending}
            isStopping={stopMutation.isPending}
            isStarting={startMutation.isPending}
            isDeleting={deleteMutation.isPending}
            isDiscarding={discardMutation.isPending}
          />
        </div>
      </div>

      {/* Create Sandbox Dialog */}
      <CreateSandboxDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onCreate={handleCreate}
        isCreating={createMutation.isPending}
        recentSandboxes={sandboxes}
        existingReservedPaths={existingReservedPaths}
        defaultProjectRoot={healthQuery.data?.config?.projectRoot}
      />

      {/* Settings Dialog */}
      <SettingsDialog
        open={settingsDialogOpen}
        onOpenChange={setSettingsDialogOpen}
      />

      {/* Commit Pending Dialog */}
      <CommitPendingDialog
        open={commitDialogOpen}
        onOpenChange={setCommitDialogOpen}
        projectRoot={healthQuery.data?.config?.projectRoot}
      />

      {/* Launch Agent Dialog */}
      {currentSandbox && (
        <LaunchAgentDialog
          open={launchDialogOpen}
          onOpenChange={setLaunchDialogOpen}
          sandbox={currentSandbox}
          onLaunch={handleLaunch}
          isLaunching={execMutation.isPending || startProcessMutation.isPending}
        />
      )}

      {/* Error Toast */}
      {(createMutation.error ||
        deleteMutation.error ||
        stopMutation.error ||
        startMutation.error ||
        approveMutation.error ||
        rejectMutation.error ||
        discardMutation.error ||
        execMutation.error ||
        startProcessMutation.error) && (
        <div
          className="fixed bottom-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md z-50"
          data-testid={SELECTORS.errorToast}
        >
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">
            {(
              createMutation.error ||
              deleteMutation.error ||
              stopMutation.error ||
              startMutation.error ||
              approveMutation.error ||
              rejectMutation.error ||
              discardMutation.error ||
              execMutation.error ||
              startProcessMutation.error
            )?.message}
          </p>
        </div>
      )}
    </div>
  );
}
