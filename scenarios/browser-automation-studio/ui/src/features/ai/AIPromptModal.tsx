import { useState, useId } from "react";
import { X, Sparkles, Wand2, FileCode, Loader, Wrench } from "lucide-react";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import toast from "react-hot-toast";
import { ResponsiveDialog } from "@shared/layout";
import { selectors } from "@constants/selectors";

interface AIPromptModalProps {
  onClose: () => void;
  folder: string;
  projectId?: string;
  onSwitchToManual?: () => void;
  onSuccess?: (workflow: Workflow) => Promise<void> | void;
}

const examplePrompts = [
  "Navigate to example.com, click the login button, fill in username and password, then take a screenshot",
  "Go to a news website, extract all article headlines and their URLs",
  "Search for 'automation' on Google, click the first result, take a full page screenshot",
  "Fill out a contact form with test data and submit it",
  "Monitor a product page for price changes every hour",
];

function AIPromptModal({
  onClose,
  folder,
  projectId,
  onSwitchToManual,
  onSuccess,
}: AIPromptModalProps) {
  const [prompt, setPrompt] = useState("");
  const [workflowName, setWorkflowName] = useState("");
  const [isGenerating, setIsGenerating] = useState(false);
  const { generateWorkflow } = useWorkflowStore();
  const titleId = useId();

  const formatErrorMessage = (message: string) => {
    if (!message) return "Failed to generate workflow";

    // Replace escaped quotes and underscores that come back from the API so the
    // guidance reads like a normal sentence for the user.
    const decoded = message.replace(/\\"/g, '"');
    const hasUnderscores =
      /[A-Z][A-Z_]+/.test(decoded) || decoded.includes("_");
    return hasUnderscores ? decoded.replace(/_/g, " ") : decoded;
  };

  const handleGenerate = async () => {
    if (!prompt.trim()) {
      toast.error("Please enter a prompt");
      return;
    }

    if (!workflowName.trim()) {
      toast.error("Please enter a workflow name");
      return;
    }

    setIsGenerating(true);
    try {
      const workflow = await generateWorkflow(
        prompt,
        workflowName,
        folder,
        projectId,
      );
      const nodeCount = Array.isArray(workflow?.nodes)
        ? workflow.nodes.length
        : 0;
      if (nodeCount === 0) {
        throw new Error(
          "AI did not return any workflow steps. Please refine your prompt and try again.",
        );
      }
      toast.success("Workflow generated successfully!");
      if (onSuccess) {
        await Promise.resolve(onSuccess(workflow));
      }
      onClose();
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to generate workflow";
      toast.error(formatErrorMessage(message));
    } finally {
      setIsGenerating(false);
    }
  };

  const handleExampleClick = (example: string) => {
    setPrompt(example);
    const name = example
      .split(",")[0]
      .toLowerCase()
      .replace(/\s+/g, "-")
      .slice(0, 30);
    setWorkflowName(name);
  };

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="wide"
      className="bg-flow-node border border-gray-700 shadow-2xl"
      data-testid={selectors.ai.modal.root}
    >
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
            <Sparkles size={20} className="text-white" />
          </div>
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-white">
              AI Workflow Generator
            </h2>
            <p className="text-xs text-gray-400">
              Describe what you want to automate
            </p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="toolbar-button p-2"
          aria-label="Close AI Workflow Generator"
        >
          <X size={18} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Manual Builder Option */}
        <div className="bg-gradient-to-r from-blue-500/10 to-purple-500/10 border border-blue-500/30 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center">
                <Wrench size={16} className="text-blue-400" />
              </div>
              <div>
                <h3 className="text-sm font-medium text-white">
                  Prefer to build manually?
                </h3>
                <p className="text-xs text-gray-400">
                  Use the visual workflow builder with drag-and-drop nodes
                </p>
              </div>
            </div>
            <button
              data-testid={selectors.ai.modal.switchToManualButton}
              onClick={onSwitchToManual}
              className="px-3 py-1.5 bg-blue-500 text-white text-sm rounded-lg hover:bg-blue-600 transition-colors flex items-center gap-2"
            >
              <Wrench size={14} />
              Manual Builder
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Workflow Name
          </label>
          <input
            type="text"
            placeholder="e.g., checkout-flow-test"
            className="w-full px-3 py-2 bg-flow-bg rounded-lg text-sm border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
            data-testid={selectors.ai.modal.workflowNameInput}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Describe Your Workflow
          </label>
          <textarea
            placeholder="Describe the browser automation steps you want to perform..."
            className="w-full px-3 py-2 bg-flow-bg rounded-lg text-sm border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
            rows={6}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            data-testid={selectors.ai.modal.promptInput}
          />
        </div>

        <div>
          <div className="flex items-center gap-2 mb-3">
            <Wand2 size={16} className="text-purple-400" />
            <span className="text-sm font-medium text-gray-300">
              Example Prompts
            </span>
          </div>
          <div className="space-y-2">
            {examplePrompts.map((example, index) => (
              <button
                key={index}
                data-testid={selectors.ai.modal.promptExample({ index })}
                onClick={() => handleExampleClick(example)}
                className="w-full text-left p-3 bg-flow-bg rounded-lg border border-gray-700 hover:border-purple-500/50 transition-colors group"
              >
                <div className="flex items-start gap-2">
                  <FileCode
                    size={14}
                    className="text-gray-500 group-hover:text-purple-400 mt-0.5"
                  />
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
                <span className="font-medium text-gray-300">Pro tip:</span> Be
                specific about selectors, wait conditions, and data you want to
                extract. The AI will generate a visual workflow you can further
                customize.
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between p-4 border-t border-gray-700">
        <div className="text-xs text-gray-500">
          Saving to: <span className="text-gray-400 font-medium">{folder}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            data-testid={selectors.ai.modal.generateButton}
            onClick={handleGenerate}
            disabled={isGenerating}
            className="px-4 py-2 bg-gradient-to-r from-purple-500 to-pink-500 text-white rounded-lg text-sm font-medium hover:from-purple-600 hover:to-pink-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {isGenerating ? (
              <>
                <Loader size={14} className="animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Sparkles size={14} />
                Generate Workflow
              </>
            )}
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default AIPromptModal;
