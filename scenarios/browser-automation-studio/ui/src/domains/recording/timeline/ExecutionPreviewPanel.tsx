/**
 * ExecutionPreviewPanel - Content-only execution viewer
 *
 * This component renders ONLY the execution content (status, progress, slideshow, live stream).
 * It is completely agnostic to presentation styling - that is handled by PreviewContainer.
 *
 * The component:
 * - Loads and displays execution status/progress
 * - Shows live stream during running executions
 * - Shows slideshow for completed executions with screenshots
 * - Exposes metadata (workflowName, currentUrl) via callbacks for the parent to use
 * - Fills available space (h-full w-full)
 */

import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { Play, Loader2, CheckCircle, XCircle, AlertCircle, RefreshCw } from 'lucide-react';
import { useExecutionStore, type Execution, useExecutionEvents } from '@/domains/executions';
import { useExecutionFrameStream } from '@/domains/executions/hooks/useExecutionFrameStream';
import { useWorkflowStore } from '@stores/workflowStore';
import {
  PlaybackControls,
  ScreenshotSlideshow,
  useSlideshowPlayback,
  ExecutionCompletionActions,
  type Screenshot,
} from '../shared';

interface ExecutionPreviewPanelProps {
  /** Execution ID to display */
  executionId: string;
  /** Callback when execution starts/restarts */
  onExecutionStart?: () => void;
  /** Callback when workflow name is determined (for BrowserChrome display) */
  onWorkflowNameChange?: (name: string | null) => void;
  /** Callback when current URL changes (for BrowserChrome display) */
  onCurrentUrlChange?: (url: string) => void;
  /** Render prop for footer content (playback controls) - passed to parent's footer */
  renderFooter?: (footer: ReactNode) => void;
  // Completion action callbacks
  /** Callback when Export button is clicked */
  onExport?: () => void;
  /** Callback when Re-run button is clicked */
  onRerun?: () => void;
  /** Callback when Edit Workflow button is clicked */
  onEditWorkflow?: () => void;
  /** Whether an export is currently in progress */
  isExporting?: boolean;
  /** Whether export is available (timeline has frames) */
  canExport?: boolean;
  /** Whether re-run is available (workflow exists) */
  canRerun?: boolean;
  /** Whether edit workflow is available (workflow + project exist) */
  canEditWorkflow?: boolean;
}

export function ExecutionPreviewPanel({
  executionId,
  onExecutionStart,
  onWorkflowNameChange,
  onCurrentUrlChange,
  renderFooter,
  onExport,
  onRerun,
  onEditWorkflow,
  isExporting,
  canExport,
  canRerun,
  canEditWorkflow,
}: ExecutionPreviewPanelProps) {
  const currentExecution = useExecutionStore((s) => s.currentExecution);
  const loadExecution = useExecutionStore((s) => s.loadExecution);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Subscribe to WebSocket updates for real-time execution progress
  useExecutionEvents(
    currentExecution ? { id: currentExecution.id, status: currentExecution.status } : undefined
  );

  // Subscribe to live frame streaming immediately when we have an execution ID
  // We subscribe early (before store loads) to avoid missing frames due to race conditions
  // The backend only sends frames when execution is actually running, so early subscription is safe
  // We unsubscribe when execution is completed/failed/cancelled
  const isExecutionRunning = currentExecution?.status === 'running';
  const isExecutionActive = !currentExecution ||
    currentExecution.status === 'pending' ||
    currentExecution.status === 'running';
  const { frameUrl, isStreaming, frameCount } = useExecutionFrameStream(
    isExecutionActive ? executionId : null,
    { enabled: isExecutionActive }
  );

  // Determine content type based on execution state and available data
  type ContentType = 'live-stream' | 'video' | 'slideshow' | 'status' | 'pending';
  const contentType: ContentType = useMemo(() => {
    // Show live stream if we're streaming frames, regardless of store state
    // This avoids showing fallback content while store is still loading
    if (isStreaming && frameUrl) return 'live-stream';
    if (!currentExecution) return 'pending';
    // Also show live stream if execution is running even if no frames yet
    if (isExecutionRunning) return 'live-stream';
    if (currentExecution.status === 'completed' || currentExecution.status === 'failed' || currentExecution.status === 'cancelled') {
      if (currentExecution.screenshots && currentExecution.screenshots.length > 0) return 'slideshow';
      return 'status';
    }
    return 'status';
  }, [currentExecution, isExecutionRunning, isStreaming, frameUrl]);

  // Get workflow info for display
  const workflows = useWorkflowStore((s) => s.workflows);
  const workflowName = currentExecution?.workflowId
    ? workflows.find((w) => w.id === currentExecution.workflowId)?.name ?? null
    : null;

  // Notify parent of workflow name changes
  useEffect(() => {
    onWorkflowNameChange?.(workflowName);
  }, [workflowName, onWorkflowNameChange]);

  // Derive current URL from the latest timeline entry
  const currentUrl = useMemo(() => {
    if (!currentExecution?.timeline?.length) return '';
    const lastEntry = currentExecution.timeline[currentExecution.timeline.length - 1];
    return lastEntry?.finalUrl || '';
  }, [currentExecution?.timeline]);

  // Notify parent of URL changes
  useEffect(() => {
    onCurrentUrlChange?.(currentUrl);
  }, [currentUrl, onCurrentUrlChange]);

  // Convert execution screenshots to slideshow format
  const slideshowScreenshots: Screenshot[] = useMemo(() => {
    if (!currentExecution?.screenshots || currentExecution.screenshots.length === 0) return [];
    return currentExecution.screenshots.map((s, i) => ({
      url: s.url,
      timestamp: s.timestamp instanceof Date ? s.timestamp.getTime() :
        (typeof s.timestamp === 'number' ? s.timestamp : undefined),
      stepLabel: s.stepName || currentExecution.timeline?.[i]?.stepType || `Step ${i + 1}`,
    }));
  }, [currentExecution?.screenshots, currentExecution?.timeline]);

  // Slideshow playback state
  const slideshow = useSlideshowPlayback(slideshowScreenshots.length, {
    interval: 1500,
    loop: true,
  });

  // Notify parent of footer content (playback controls)
  useEffect(() => {
    if (contentType === 'slideshow' && slideshowScreenshots.length > 1) {
      renderFooter?.(
        <PlaybackControls
          contentType="slideshow"
          isPlaying={slideshow.isPlaying}
          progress={slideshow.progress}
          currentIndex={slideshow.currentIndex}
          totalFrames={slideshow.totalFrames}
          onPlayPause={slideshow.toggle}
          onPrevious={slideshow.previous}
          onNext={slideshow.next}
          onSeek={(position) => {
            slideshow.seekTo(Math.round(position * (slideshow.totalFrames - 1)));
          }}
        />
      );
    } else {
      renderFooter?.(null);
    }
  }, [contentType, slideshowScreenshots.length, slideshow, renderFooter]);

  const handleRetry = useCallback(() => {
    setError(null);
    setIsLoading(true);
    loadExecution(executionId).catch((err) => {
      setError(err instanceof Error ? err.message : 'Failed to load execution');
      setIsLoading(false);
    });
  }, [executionId, loadExecution]);

  // Load execution data on mount
  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError(null);
      try {
        await loadExecution(executionId);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load execution');
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [executionId, loadExecution]);

  // Loading state - but show live stream if frames are arriving even while loading
  if (isLoading && !(isStreaming && frameUrl)) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center text-center px-6">
        <Loader2 className="w-12 h-12 text-flow-accent animate-spin mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Loading execution...
        </p>
      </div>
    );
  }

  // Error state - but still show live stream if frames are arriving
  if (error && !(isStreaming && frameUrl)) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center text-center px-6">
        <AlertCircle className="w-12 h-12 text-red-500 mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Failed to load execution
        </p>
        <p className="text-xs text-gray-500 mt-1">{error}</p>
        <button
          onClick={handleRetry}
          className="mt-4 flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors"
        >
          <RefreshCw size={16} />
          Retry
        </button>
      </div>
    );
  }

  // No execution found - but still show live stream if frames are arriving
  if (!currentExecution && !(isStreaming && frameUrl)) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center text-center px-6">
        <AlertCircle className="w-12 h-12 text-amber-500 mb-4" />
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Execution not found
        </p>
        <p className="text-xs text-gray-500 mt-1">
          The execution may have been deleted or expired
        </p>
      </div>
    );
  }

  // Main content
  return (
    <div className="h-full w-full">
      {contentType === 'live-stream' && frameUrl ? (
        <LiveStreamView frameUrl={frameUrl} frameCount={frameCount} />
      ) : contentType === 'slideshow' && currentExecution ? (
        <ScreenshotSlideshow
          screenshots={slideshowScreenshots}
          currentIndex={slideshow.currentIndex}
        />
      ) : currentExecution ? (
        <ExecutionContent
          execution={currentExecution}
          onStart={onExecutionStart}
          onExport={onExport}
          onRerun={onRerun}
          onEditWorkflow={onEditWorkflow}
          isExporting={isExporting}
          canExport={canExport}
          canRerun={canRerun}
          canEditWorkflow={canEditWorkflow}
        />
      ) : (
        // Fallback for edge case where we have no execution and no streaming
        <div className="h-full w-full flex flex-col items-center justify-center text-center px-6">
          <Loader2 className="w-12 h-12 text-flow-accent animate-spin mb-4" />
          <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
            Waiting for execution...
          </p>
        </div>
      )}
    </div>
  );
}

/** Main execution content based on status */
function ExecutionContent({
  execution,
  onStart,
  onExport,
  onRerun,
  onEditWorkflow,
  isExporting,
  canExport,
  canRerun,
  canEditWorkflow,
}: {
  execution: Execution;
  onStart?: () => void;
  onExport?: () => void;
  onRerun?: () => void;
  onEditWorkflow?: () => void;
  isExporting?: boolean;
  canExport?: boolean;
  canRerun?: boolean;
  canEditWorkflow?: boolean;
}) {
  const { status, progress, currentStep, error, screenshots, timeline } = execution;

  // Running state - show progress
  if (status === 'running') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        {/* Progress circle */}
        <div className="relative w-32 h-32 mb-6">
          <svg className="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
            <circle
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke="currentColor"
              strokeWidth="8"
              className="text-gray-200 dark:text-gray-700"
            />
            <circle
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke="currentColor"
              strokeWidth="8"
              strokeLinecap="round"
              className="text-flow-accent"
              strokeDasharray={`${(progress || 0) * 2.83} 283`}
            />
          </svg>
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {Math.round(progress || 0)}%
            </span>
          </div>
        </div>

        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">
          Execution in progress
        </p>
        {currentStep && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {currentStep}
          </p>
        )}

        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <p className="text-xs text-gray-400 dark:text-gray-500 mb-2 uppercase tracking-wide font-medium">
              Latest Screenshot
            </p>
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Latest step screenshot"
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm"
            />
          </div>
        )}
      </div>
    );
  }

  // Completed state
  if (status === 'completed') {
    const stepCount = timeline?.length || 0;
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <CheckCircle className="w-16 h-16 text-green-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Completed
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {stepCount} step{stepCount !== 1 ? 's' : ''} executed successfully
        </p>

        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Final screenshot"
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm"
            />
          </div>
        )}

        <ExecutionCompletionActions
          onExport={onExport}
          onRerun={onRerun}
          onEditWorkflow={onEditWorkflow}
          isExporting={isExporting}
          canExport={canExport}
          canRerun={canRerun}
          canEditWorkflow={canEditWorkflow}
        />
      </div>
    );
  }

  // Failed state
  if (status === 'failed') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <XCircle className="w-16 h-16 text-red-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Failed
        </p>
        {error && (
          <p className="text-sm text-red-500 dark:text-red-400 mt-2 max-w-md">
            {error}
          </p>
        )}
        {currentStep && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
            Failed at: {currentStep}
          </p>
        )}

        {screenshots && screenshots.length > 0 && (
          <div className="mt-6 w-full max-w-md">
            <p className="text-xs text-gray-400 dark:text-gray-500 mb-2 uppercase tracking-wide font-medium">
              Screenshot at Failure
            </p>
            <img
              src={screenshots[screenshots.length - 1].url}
              alt="Failure screenshot"
              className="w-full rounded-lg border border-red-200 dark:border-red-800 shadow-sm"
            />
          </div>
        )}

        <ExecutionCompletionActions
          onExport={onExport}
          onRerun={onRerun}
          onEditWorkflow={onEditWorkflow}
          isExporting={isExporting}
          canExport={canExport}
          canRerun={canRerun}
          canEditWorkflow={canEditWorkflow}
        />
      </div>
    );
  }

  // Cancelled state
  if (status === 'cancelled') {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <XCircle className="w-16 h-16 text-amber-500 mb-4" />
        <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
          Execution Cancelled
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          The execution was stopped before completion
        </p>

        <ExecutionCompletionActions
          onExport={onExport}
          onRerun={onRerun}
          onEditWorkflow={onEditWorkflow}
          isExporting={isExporting}
          canExport={canExport}
          canRerun={canRerun}
          canEditWorkflow={canEditWorkflow}
        />
      </div>
    );
  }

  // Pending/default state
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      <div className="w-20 h-20 rounded-full bg-flow-accent/10 flex items-center justify-center mb-6">
        <Play className="w-10 h-10 text-flow-accent" />
      </div>
      <p className="text-lg font-medium text-gray-800 dark:text-gray-200">
        Ready to Execute
      </p>
      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm">
        The workflow is ready to run. Click the button below to start execution.
      </p>

      {onStart && (
        <button
          onClick={onStart}
          className="mt-6 flex items-center gap-2 px-6 py-3 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors shadow-lg shadow-flow-accent/25"
        >
          <Play size={18} />
          Start Execution
        </button>
      )}
    </div>
  );
}

/** Live stream view - displays real-time frames from execution */
function LiveStreamView({
  frameUrl,
  frameCount,
}: {
  frameUrl: string;
  frameCount: number;
}) {
  return (
    <div className="h-full w-full flex flex-col items-center justify-center bg-gray-900">
      <div className="relative w-full h-full flex items-center justify-center">
        <img
          src={frameUrl}
          alt="Live execution preview"
          className="max-w-full max-h-full object-contain"
        />
        <div className="absolute top-3 left-3 flex items-center gap-2 px-2 py-1 bg-red-600/90 rounded text-white text-xs font-medium">
          <span className="w-2 h-2 bg-white rounded-full animate-pulse" />
          LIVE
        </div>
        <div className="absolute bottom-3 right-3 px-2 py-1 bg-black/60 rounded text-white text-xs">
          Frame {frameCount}
        </div>
      </div>
    </div>
  );
}
