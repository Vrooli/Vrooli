/**
 * InlineExecutionViewer Component
 *
 * A compact execution viewer for the Project view that reuses components from
 * the Record page. Displays execution with two tabs: Replay and Artifacts.
 */

import { useState, useCallback, useMemo, useEffect } from 'react';
import { Play, FileText } from 'lucide-react';
import { InlineExecutionHeader } from './InlineExecutionHeader';
import { ExecutionPreviewPanel } from '@/domains/recording/timeline/ExecutionPreviewPanel';
import { ArtifactsTab } from '@/domains/recording/sidebar/ArtifactsTab';
import type { ConsoleLogEntry, NetworkEventEntry, DomSnapshotEntry } from '@/domains/recording/sidebar/ArtifactsTab';
import type { ArtifactSubType } from '@/domains/recording/sidebar/types';
import { useExecutionStore, type TimelineFrame, type Execution } from './store';
import { useWorkflowStore } from '@stores/workflowStore';
import { useExecutionExport } from './viewer/useExecutionExport';
import { useReplayCustomization } from './viewer/useReplayCustomization';
import { useExportStore } from '@/domains/exports';
import ExportDialog from './viewer/ExportDialog';
import { useExecutionEvents } from './hooks/useExecutionEvents';

type ViewerTab = 'replay' | 'artifacts';

export interface InlineExecutionViewerProps {
  /** Execution ID to display */
  executionId: string;
  /** Workflow ID (for re-run and metadata) */
  workflowId: string;
  /** Project ID (for navigation - kept for future use) */
  projectId: string;
  /** Callback when close is clicked */
  onClose: () => void;
  /** Callback when re-run is requested (optional, if not provided re-run is disabled) */
  onRerun?: () => void;
}

// Default execution for export hook when no execution loaded
const defaultExecution: Execution = {
  id: '',
  workflowId: '',
  status: 'pending',
  screenshots: [],
  logs: [],
  timeline: [],
  progress: 0,
  startedAt: new Date(),
};

// ============================================================================
// Data Extraction Helpers
// ============================================================================

function extractConsoleLogs(timeline: TimelineFrame[]): ConsoleLogEntry[] {
  const consoleLogs: ConsoleLogEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'console_log' && artifact.payload) {
        const payload = artifact.payload as {
          level?: string;
          text?: string;
          timestamp?: string;
          stack?: string;
          location?: string;
        };
        consoleLogs.push({
          id: artifact.id,
          level: (payload.level as ConsoleLogEntry['level']) ?? 'log',
          text: payload.text ?? '',
          timestamp: payload.timestamp ? new Date(payload.timestamp) : new Date(),
          stack: payload.stack,
          location: payload.location,
        });
      }
    }
  }
  return consoleLogs;
}

function extractNetworkEvents(timeline: TimelineFrame[]): NetworkEventEntry[] {
  const networkEvents: NetworkEventEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'network_event' && artifact.payload) {
        const payload = artifact.payload as {
          type?: string;
          url?: string;
          method?: string;
          resourceType?: string;
          status?: number;
          ok?: boolean;
          failure?: string;
          timestamp?: string;
          durationMs?: number;
        };
        networkEvents.push({
          id: artifact.id,
          type: (payload.type as NetworkEventEntry['type']) ?? 'request',
          url: payload.url ?? '',
          method: payload.method,
          resourceType: payload.resourceType,
          status: payload.status,
          ok: payload.ok,
          failure: payload.failure,
          timestamp: payload.timestamp ? new Date(payload.timestamp) : new Date(),
          durationMs: payload.durationMs,
        });
      }
    }
  }
  return networkEvents;
}

function extractDomSnapshots(timeline: TimelineFrame[]): DomSnapshotEntry[] {
  const snapshots: DomSnapshotEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'dom_snapshot') {
        snapshots.push({
          id: artifact.id,
          stepIndex: artifact.step_index ?? frame.stepIndex ?? 0,
          stepName: artifact.label ?? frame.stepType ?? 'Step',
          timestamp: new Date(),
        });
      }
    }
  }
  return snapshots;
}

// ============================================================================
// Component
// ============================================================================

export function InlineExecutionViewer({
  executionId,
  workflowId,
  projectId: _projectId, // Kept for future navigation features
  onClose,
  onRerun,
}: InlineExecutionViewerProps) {
  const [activeTab, setActiveTab] = useState<ViewerTab>('replay');
  const [selectedScreenshotIndex, setSelectedScreenshotIndex] = useState<number>(0);
  const [logFilter, setLogFilter] = useState<'all' | 'error' | 'warning' | 'info' | 'success'>('all');
  const [artifactSubType, setArtifactSubType] = useState<ArtifactSubType>('screenshots');
  const [isRerunning, setIsRerunning] = useState(false);

  // Get execution from store
  const currentExecution = useExecutionStore((s) => s.currentExecution);
  const autoOpenExport = useExecutionStore((s) => s.autoOpenExport);
  const setAutoOpenExport = useExecutionStore((s) => s.setAutoOpenExport);

  // Subscribe to WebSocket updates for real-time progress
  useExecutionEvents(
    currentExecution ? { id: currentExecution.id, status: currentExecution.status } : undefined
  );

  // Get workflow name
  const workflows = useWorkflowStore((s) => s.workflows);
  const workflowName = useMemo(() => {
    return workflows.find((w) => w.id === workflowId)?.name ?? 'Workflow';
  }, [workflows, workflowId]);

  // Export setup
  const { createExport } = useExportStore();
  const replayCustomization = useReplayCustomization({ executionId });
  const exportController = useExecutionExport({
    execution: currentExecution ?? defaultExecution,
    replayFrames: currentExecution?.timeline ?? [],
    workflowName,
    replayCustomization,
    createExport: createExport as Parameters<typeof useExecutionExport>[0]['createExport'],
  });

  // Handle re-run
  const handleRerun = useCallback(async () => {
    if (!onRerun) return;
    setIsRerunning(true);
    try {
      await onRerun();
    } finally {
      setIsRerunning(false);
    }
  }, [onRerun]);

  // Extract artifacts data
  const timeline = currentExecution?.timeline ?? [];
  const consoleLogs = useMemo(() => extractConsoleLogs(timeline), [timeline]);
  const networkEvents = useMemo(() => extractNetworkEvents(timeline), [timeline]);
  const domSnapshots = useMemo(() => extractDomSnapshots(timeline), [timeline]);

  const canExport = (currentExecution?.timeline?.length ?? 0) > 0;

  // Auto-open export dialog when coming from the Exports tab "New Export" flow
  useEffect(() => {
    if (autoOpenExport && canExport && !exportController.isExportDialogOpen) {
      exportController.openExportDialog();
      setAutoOpenExport(false);
    }
  }, [autoOpenExport, canExport, exportController, setAutoOpenExport]);

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900">
      {/* Header */}
      <InlineExecutionHeader
        workflowName={workflowName}
        status={currentExecution?.status}
        onRerun={onRerun ? handleRerun : undefined}
        onExport={canExport ? exportController.openExportDialog : undefined}
        onClose={onClose}
        canRerun={!!onRerun}
        canExport={canExport}
        isExporting={exportController.isExporting}
        isRerunning={isRerunning}
      />

      {/* Tab Bar */}
      <div className="flex border-b border-gray-200 dark:border-gray-700">
        <button
          type="button"
          onClick={() => setActiveTab('replay')}
          className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
            activeTab === 'replay'
              ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-b-2 border-blue-500'
              : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-800'
          }`}
        >
          <Play size={14} />
          Replay
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('artifacts')}
          className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
            activeTab === 'artifacts'
              ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-b-2 border-blue-500'
              : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-800'
          }`}
        >
          <FileText size={14} />
          Artifacts
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'replay' ? (
          <ExecutionPreviewPanel
            executionId={executionId}
            onExport={canExport ? exportController.openExportDialog : undefined}
            onRerun={onRerun ? handleRerun : undefined}
            canExport={canExport}
            canRerun={!!onRerun}
            isExporting={exportController.isExporting}
          />
        ) : (
          <ArtifactsTab
            screenshots={currentExecution?.screenshots ?? []}
            selectedScreenshotIndex={selectedScreenshotIndex}
            onSelectScreenshot={setSelectedScreenshotIndex}
            executionLogs={currentExecution?.logs ?? []}
            logFilter={logFilter}
            onLogFilterChange={setLogFilter}
            consoleLogs={consoleLogs}
            networkEvents={networkEvents}
            domSnapshots={domSnapshots}
            executionStatus={currentExecution?.status}
            activeSubType={artifactSubType}
            onSubTypeChange={setArtifactSubType}
            className="h-full"
          />
        )}
      </div>

      {/* Export Dialog */}
      <ExportDialog
        isOpen={exportController.isExportDialogOpen}
        onClose={exportController.closeExportDialog}
        onConfirm={exportController.confirmExport}
        dialogTitleId="inline-export-dialog-title"
        dialogDescriptionId="inline-export-dialog-description"
        {...exportController.exportDialogProps}
      />
    </div>
  );
}
