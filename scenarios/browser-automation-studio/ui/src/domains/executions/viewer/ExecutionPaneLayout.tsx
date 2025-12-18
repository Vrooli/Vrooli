/**
 * Execution Pane Layout
 *
 * Handles the resizable execution viewer pane that appears alongside the workflow builder.
 * Extracts resize logic that was previously in App.tsx.
 */
import { useState, useCallback, useEffect, lazy, Suspense, type ReactNode } from "react";
import { LoadingSpinner } from "@shared/ui";
import type { Execution } from "../store";

const ExecutionViewer = lazy(() => import("./ExecutionViewer"));
const ResponsiveDialog = lazy(() => import("@shared/layout/ResponsiveDialog"));

// Pane dimension constants
const EXECUTION_MIN_WIDTH = 360;
const EXECUTION_MAX_WIDTH = 720;
const EXECUTION_DEFAULT_WIDTH = 440;

interface ExecutionPaneLayoutProps {
  /** Whether the execution viewer should be shown */
  isOpen: boolean;
  /** The workflow ID for the execution viewer */
  workflowId: string | null;
  /** Current execution data */
  execution: Execution | null;
  /** Whether on a large screen (desktop) */
  isLargeScreen: boolean;
  /** Callback to close the viewer */
  onClose: () => void;
  /** The main content to render (workflow builder) */
  children: ReactNode;
}

interface ResizeState {
  width: number;
  isResizing: boolean;
}

/**
 * Layout component that manages the resizable execution pane.
 * On large screens, shows a side panel with drag-to-resize.
 * On small screens, shows a modal dialog.
 */
export function ExecutionPaneLayout({
  isOpen,
  workflowId,
  execution,
  isLargeScreen,
  onClose,
  children,
}: ExecutionPaneLayoutProps) {
  const [resizeState, setResizeState] = useState<ResizeState>({
    width: EXECUTION_DEFAULT_WIDTH,
    isResizing: false,
  });

  const clampWidth = useCallback(
    (value: number) =>
      Math.min(EXECUTION_MAX_WIDTH, Math.max(EXECUTION_MIN_WIDTH, value)),
    []
  );

  // Clean up resize state when pane closes
  useEffect(() => {
    if (!isOpen) {
      setResizeState((prev) => ({ ...prev, isResizing: false }));
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    }
  }, [isOpen]);

  const handleResizeMouseDown = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      if (!isLargeScreen) {
        return;
      }

      event.preventDefault();
      event.stopPropagation();

      const startX = event.clientX;
      const startWidth = resizeState.width;

      setResizeState((prev) => ({ ...prev, isResizing: true }));
      document.body.style.userSelect = "none";
      document.body.style.cursor = "col-resize";

      const onMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const nextWidth = clampWidth(startWidth - deltaX);
        setResizeState((prev) => ({ ...prev, width: nextWidth }));
      };

      const onMouseUp = () => {
        setResizeState((prev) => ({ ...prev, isResizing: false }));
        document.body.style.userSelect = "";
        document.body.style.cursor = "";
        window.removeEventListener("mousemove", onMouseMove);
        window.removeEventListener("mouseup", onMouseUp);
      };

      window.addEventListener("mousemove", onMouseMove);
      window.addEventListener("mouseup", onMouseUp);
    },
    [clampWidth, resizeState.width, isLargeScreen]
  );

  // Render the execution viewer content
  const renderExecutionViewer = () => (
    <Suspense
      fallback={
        <div className="h-full flex items-center justify-center">
          <LoadingSpinner variant="minimal" size={20} />
        </div>
      }
    >
      <ExecutionViewer
        workflowId={workflowId!}
        execution={execution}
        onClose={onClose}
        showExecutionSwitcher
      />
    </Suspense>
  );

  return (
    <>
      <div
        className={`flex-1 flex min-h-0 ${resizeState.isResizing ? "select-none" : ""}`}
      >
        {/* Main content (workflow builder) */}
        <div className="flex-1 flex flex-col min-h-0">
          {children}
        </div>

        {/* Desktop: Side panel with resize handle */}
        {isOpen && workflowId && isLargeScreen && (
          <>
            {/* Resize handle */}
            <div
              role="separator"
              aria-orientation="vertical"
              className={`w-1 cursor-col-resize transition-colors ${
                resizeState.isResizing
                  ? "bg-flow-accent/50"
                  : "bg-transparent hover:bg-flow-accent/40"
              }`}
              onMouseDown={handleResizeMouseDown}
              aria-label="Resize execution viewer pane"
            />
            {/* Execution viewer pane */}
            <div
              className="border-l border-gray-800 flex flex-col min-h-0"
              style={{
                width: resizeState.width,
                minWidth: EXECUTION_MIN_WIDTH,
              }}
            >
              {renderExecutionViewer()}
            </div>
          </>
        )}
      </div>

      {/* Mobile: Modal dialog */}
      {isOpen && workflowId && !isLargeScreen && (
        <Suspense
          fallback={
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <LoadingSpinner variant="minimal" size={20} />
            </div>
          }
        >
          <ResponsiveDialog
            isOpen={true}
            onDismiss={onClose}
            ariaLabel="Execution Viewer"
            size="xl"
          >
            {renderExecutionViewer()}
          </ResponsiveDialog>
        </Suspense>
      )}
    </>
  );
}

export default ExecutionPaneLayout;
