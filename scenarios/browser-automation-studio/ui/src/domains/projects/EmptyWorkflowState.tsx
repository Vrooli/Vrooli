import {
  FileCode,
  WifiOff,
  FolderTree,
  Circle,
  Sparkles,
  Upload,
} from "lucide-react";

interface EmptyWorkflowStateProps {
  error?: string | null;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
  onStartRecording?: () => void;
  onImportWorkflow?: () => void;
}

/**
 * Empty state component for when no workflows exist in a project.
 * Shows a welcoming message with quick action cards for creating workflows.
 * Used by both WorkflowCardGrid (card view) and ProjectFileTree (tree view).
 */
export function EmptyWorkflowState({
  error,
  onCreateWorkflow,
  onCreateWorkflowDirect,
  onStartRecording,
  onImportWorkflow,
}: EmptyWorkflowStateProps) {
  return (
    <div className="flex-1 overflow-auto p-6">
      <div className="max-w-2xl mx-auto pt-8">
        <div className="text-center mb-8">
          <div className="mb-5 flex items-center justify-center">
            <div className="p-4 bg-green-500/20 rounded-2xl">
              {error ? (
                <WifiOff size={40} className="text-red-400" />
              ) : (
                <FileCode size={40} className="text-green-400" />
              )}
            </div>
          </div>
          <h3 className="text-xl font-semibold text-surface mb-2">
            {error ? "Unable to Load Workflows" : "Ready to Automate"}
          </h3>
          <p className="text-gray-400 max-w-md mx-auto">
            {error
              ? "There was an issue connecting to the API. You can still use the interface when the connection is restored."
              : "Create your first workflow to automate browser tasks. Use AI to describe what you want, or build visually with the drag-and-drop builder."}
          </p>
        </div>

        {!error && (
          <>
            {/* Quick Actions - Order: Record, AI-Assisted, Visual Builder, Import */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
              {/* Record - Primary workflow creation method */}
              {onStartRecording && (
                <button
                  onClick={onStartRecording}
                  className="bg-flow-node border border-red-500/30 rounded-xl p-5 text-left hover:border-red-500/60 hover:shadow-lg hover:shadow-red-500/10 transition-all group"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 bg-red-500/20 rounded-lg group-hover:bg-red-500/30 transition-colors">
                      <Circle size={20} className="text-red-400 fill-red-400" />
                    </div>
                    <h4 className="font-medium text-surface">Record</h4>
                  </div>
                  <p className="text-sm text-gray-400">
                    Record your browser actions and convert them into a reusable workflow.
                  </p>
                </button>
              )}

              {/* AI-Assisted */}
              <button
                onClick={onCreateWorkflow}
                className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-purple-500/60 hover:shadow-lg hover:shadow-purple-500/10 transition-all group"
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                    <Sparkles size={20} className="text-purple-400" />
                  </div>
                  <h4 className="font-medium text-surface">AI-Assisted</h4>
                </div>
                <p className="text-sm text-gray-400">
                  Describe your automation in plain language and let AI generate the workflow.
                </p>
              </button>

              {/* Visual Builder */}
              <button
                onClick={onCreateWorkflowDirect ?? onCreateWorkflow}
                className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-blue-500/60 hover:shadow-lg hover:shadow-blue-500/10 transition-all group"
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                    <FolderTree size={20} className="text-blue-400" />
                  </div>
                  <h4 className="font-medium text-surface">Visual Builder</h4>
                </div>
                <p className="text-sm text-gray-400">
                  Use the drag-and-drop interface to build workflows step by step.
                </p>
              </button>

              {/* Import */}
              {onImportWorkflow && (
                <button
                  onClick={onImportWorkflow}
                  className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-green-500/60 hover:shadow-lg hover:shadow-green-500/10 transition-all group"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 bg-green-500/20 rounded-lg group-hover:bg-green-500/30 transition-colors">
                      <Upload size={20} className="text-green-400" />
                    </div>
                    <h4 className="font-medium text-surface">Import</h4>
                  </div>
                  <p className="text-sm text-gray-400">
                    Import an existing workflow file from your filesystem.
                  </p>
                </button>
              )}
            </div>

            {/* Workflow ideas */}
            <div className="text-center text-sm text-gray-500">
              <p className="mb-2">Try automating:</p>
              <div className="flex flex-wrap justify-center gap-2">
                <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Form submissions</span>
                <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">UI testing</span>
                <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Data extraction</span>
                <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Screenshots</span>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
