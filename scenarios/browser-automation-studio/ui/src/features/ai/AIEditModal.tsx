import { useState, useId } from 'react';
import { X, Sparkles, Wand2, Bug, Loader, Edit3 } from 'lucide-react';
import { useWorkflowStore } from '@stores/workflowStore';
import toast from 'react-hot-toast';
import { ResponsiveDialog } from '@shared/layout';

interface AIEditModalProps {
  onClose: () => void;
}

const examplePrompts = [
  "Add error handling to all click actions",
  "Add a 2 second wait between each action",
  "Replace all hardcoded selectors with more robust XPath selectors",
  "Add screenshot capture after each important action",
  "Optimize the workflow by removing unnecessary wait times",
  "Add data extraction to capture form values before submission",
  "Add conditional logic to handle different page states",
  "Add retry logic for failed actions",
];

function AIEditModal({ onClose }: AIEditModalProps) {
  const [prompt, setPrompt] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const { currentWorkflow, modifyWorkflow } = useWorkflowStore();
  const titleId = useId();

  const handleApplyChanges = async () => {
    if (!prompt.trim()) {
      toast.error('Please describe the changes you want to make');
      return;
    }
    
    if (!currentWorkflow) {
      toast.error('No workflow loaded to modify');
      return;
    }

    setIsProcessing(true);
    try {
      await modifyWorkflow(prompt);
      toast.success('Workflow modified successfully!');
      onClose();
    } catch (error) {
      toast.error('Failed to modify workflow');
    } finally {
      setIsProcessing(false);
    }
  };

  const handleExampleClick = (example: string) => {
    setPrompt(example);
  };

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="wide"
      className="bg-flow-node border border-gray-700 shadow-2xl"
    >
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-gradient-to-br from-orange-500 to-red-500 rounded-lg flex items-center justify-center">
            <Bug size={20} className="text-white" />
          </div>
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-white">AI Workflow Editor</h2>
            <p className="text-xs text-gray-400">Describe changes to make to your workflow</p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="toolbar-button p-2"
          aria-label="Close AI Workflow Editor"
        >
          <X size={18} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Current Workflow Info */}
        {currentWorkflow && (
          <div className="bg-flow-bg rounded-lg p-3 border border-gray-700">
            <div className="flex items-center gap-2 mb-1">
              <Edit3 size={14} className="text-blue-400" />
              <span className="text-sm font-medium text-gray-300">Editing Workflow</span>
            </div>
            <p className="text-sm text-gray-400">
              {currentWorkflow.name || 'Untitled Workflow'}
            </p>
            <p className="text-xs text-gray-500 mt-1">
              {currentWorkflow.nodes?.length || 0} nodes, {currentWorkflow.edges?.length || 0} connections
            </p>
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Describe Your Changes
          </label>
          <textarea
            placeholder="Describe the modifications you want to make to this workflow..."
            className="w-full px-3 py-2 bg-flow-bg rounded-lg text-sm border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
            rows={6}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
          />
        </div>

        <div>
          <div className="flex items-center gap-2 mb-3">
            <Wand2 size={16} className="text-orange-400" />
            <span className="text-sm font-medium text-gray-300">Common Modifications</span>
          </div>
          <div className="space-y-2">
            {examplePrompts.map((example, index) => (
              <button
                key={index}
                onClick={() => handleExampleClick(example)}
                className="w-full text-left p-3 bg-flow-bg rounded-lg border border-gray-700 hover:border-orange-500/50 transition-colors group"
              >
                <div className="flex items-start gap-2">
                  <Bug size={14} className="text-gray-500 group-hover:text-orange-400 mt-0.5" />
                  <span className="text-xs text-gray-400 group-hover:text-gray-300">
                    {example}
                  </span>
                </div>
              </button>
            ))}
          </div>
        </div>

        <div className="bg-flow-bg rounded-lg p-3 border border-gray-700">
          <div className="flex items-start gap-2">
            <Sparkles size={14} className="text-yellow-400 mt-0.5" />
            <div className="flex-1">
              <p className="text-xs text-gray-400">
                <span className="font-medium text-gray-300">Pro tip:</span> Be specific about the changes you want.
                The AI will analyze your current workflow and apply the modifications while preserving existing functionality.
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between p-4 border-t border-gray-700">
        <div className="text-xs text-gray-500">
          {currentWorkflow?.nodes?.length || 0} nodes will be analyzed and potentially modified
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleApplyChanges}
            disabled={isProcessing}
            className="px-4 py-2 bg-gradient-to-r from-orange-500 to-red-500 text-white rounded-lg text-sm font-medium hover:from-orange-600 hover:to-red-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {isProcessing ? (
              <>
                <Loader size={14} className="animate-spin" />
                Modifying...
              </>
            ) : (
              <>
                <Sparkles size={14} />
                Apply Changes
              </>
            )}
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default AIEditModal;