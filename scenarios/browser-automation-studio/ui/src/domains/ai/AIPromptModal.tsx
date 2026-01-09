import { useState, useCallback, useMemo, useId } from "react";
import { useNavigate } from "react-router-dom";
import {
  X,
  Sparkles,
  Zap,
  GitBranch,
  CheckSquare,
  Globe,
  Play,
  Lightbulb,
} from "lucide-react";
import { ResponsiveDialog } from "@shared/layout";
import { BrowserUrlBar } from "@/domains/recording";
import { selectors } from "@constants/selectors";

type WorkflowType = "action" | "flow" | "case";

interface AIPromptModalProps {
  onClose: () => void;
  folder: string;
  projectId?: string;
  onSwitchToManual?: () => void;
}

// Type configurations with descriptions and example prompts
const typeConfigs: Record<
  WorkflowType,
  {
    icon: typeof Zap;
    color: string;
    bgColor: string;
    borderColor: string;
    title: string;
    description: string;
    examples: string[];
    proTip: string;
  }
> = {
  action: {
    icon: Zap,
    color: "text-green-400",
    bgColor: "bg-green-500/20",
    borderColor: "border-green-500/30",
    title: "Action",
    description: "Single step automation - one focused interaction",
    examples: [
      "Click the login button",
      "Extract the page title",
      "Take a screenshot of the hero section",
    ],
    proTip:
      "Actions are atomic steps. Keep your prompt focused on one specific interaction.",
  },
  flow: {
    icon: GitBranch,
    color: "text-blue-400",
    bgColor: "bg-blue-500/20",
    borderColor: "border-blue-500/30",
    title: "Flow",
    description: "Multi-step sequence - chain multiple actions together",
    examples: [
      "Navigate to login, enter credentials, and click submit",
      "Search for a product, add to cart, proceed to checkout",
      "Fill out the registration form with test data",
    ],
    proTip:
      "Flows chain multiple actions. Describe the full sequence you want automated.",
  },
  case: {
    icon: CheckSquare,
    color: "text-purple-400",
    bgColor: "bg-purple-500/20",
    borderColor: "border-purple-500/30",
    title: "Case",
    description: "Test with assertions - verify expected behavior",
    examples: [
      "Login and verify the dashboard shows the welcome message",
      "Submit empty form and check error messages appear",
      "Add item to cart and verify cart count increases",
    ],
    proTip:
      'Cases include assertions. Describe what you want to verify using words like "verify", "check", "ensure".',
  },
};

function AIPromptModal({
  onClose,
  folder,
  projectId,
}: AIPromptModalProps) {
  const navigate = useNavigate();
  const titleId = useId();

  const [workflowType, setWorkflowType] = useState<WorkflowType>("flow");
  const [url, setUrl] = useState("");
  const [prompt, setPrompt] = useState("");

  const currentConfig = typeConfigs[workflowType];

  // Validation
  const { isValid, validationMessage } = useMemo(() => {
    if (!url.trim()) {
      return { isValid: false, validationMessage: "Enter a website URL to continue" };
    }
    if (!prompt.trim()) {
      return { isValid: false, validationMessage: "Describe what you want to automate" };
    }
    return { isValid: true, validationMessage: "" };
  }, [url, prompt]);

  // Handle URL navigation (from URL bar)
  const handleUrlNavigate = useCallback((navigatedUrl: string) => {
    setUrl(navigatedUrl);
  }, []);

  // Handle example click
  const handleExampleClick = useCallback((example: string) => {
    setPrompt(example);
  }, []);

  // Handle form submission - navigate to Record page with AI mode
  const handleStart = useCallback(() => {
    if (!isValid) return;

    // For case type, enhance the prompt to encourage assertions
    let finalPrompt = prompt;
    if (workflowType === "case") {
      // If prompt doesn't already have verification language, suggest adding assertions
      const hasVerificationLang = /verify|check|ensure|confirm|assert|validate|expect/i.test(prompt);
      if (!hasVerificationLang) {
        finalPrompt = `${prompt}. After completing the actions, verify the expected outcome.`;
      }
    }

    // Navigate to record page with query params
    const params = new URLSearchParams({
      url,
      ai_prompt: finalPrompt,
      workflow_type: workflowType,
    });

    if (projectId) {
      params.set("project_id", projectId);
    }
    if (folder) {
      params.set("folder", folder);
    }

    onClose();
    navigate(`/record?${params.toString()}`);
  }, [isValid, url, prompt, workflowType, projectId, folder, onClose, navigate]);

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="wide"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
      data-testid={selectors.ai.modal.root}
    >
      <div>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          <button
            onClick={onClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>

          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-purple-500/30 to-pink-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-purple-500/10">
              <Sparkles size={28} className="text-purple-400" />
            </div>
            <h2
              id={titleId}
              className="text-xl font-bold text-white tracking-tight"
            >
              AI Workflow Generator
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              Create workflows using AI-powered browser automation
            </p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6 space-y-5">
          {/* Workflow Type Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-3">
              What type of workflow?
            </label>
            <div className="grid grid-cols-3 gap-3">
              {(Object.keys(typeConfigs) as WorkflowType[]).map((type) => {
                const config = typeConfigs[type];
                const Icon = config.icon;
                const isSelected = workflowType === type;

                return (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setWorkflowType(type)}
                    className={`
                      relative flex flex-col items-center p-4 rounded-xl border-2 transition-all duration-200 text-center
                      ${
                        isSelected
                          ? `${config.borderColor} ${config.bgColor}`
                          : "border-gray-700 bg-gray-800/50 hover:border-gray-600 hover:bg-gray-800"
                      }
                    `}
                  >
                    <div
                      className={`w-10 h-10 rounded-lg flex items-center justify-center mb-2 ${
                        isSelected ? config.bgColor : "bg-gray-700"
                      }`}
                    >
                      <Icon
                        size={20}
                        className={isSelected ? config.color : "text-gray-400"}
                      />
                    </div>
                    <h3
                      className={`font-semibold text-sm ${
                        isSelected ? "text-white" : "text-gray-300"
                      }`}
                    >
                      {config.title}
                    </h3>
                    <p className="text-xs text-gray-500 mt-1 leading-tight">
                      {config.description}
                    </p>
                  </button>
                );
              })}
            </div>
          </div>

          {/* URL Input */}
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              <Globe size={14} className="inline mr-1.5" />
              Starting URL <span className="text-red-400">*</span>
            </label>
            <div className="rounded-xl border-2 border-gray-700/50 bg-gray-800/50 overflow-hidden focus-within:border-purple-500/50 transition-colors">
              <BrowserUrlBar
                value={url}
                onChange={setUrl}
                onNavigate={handleUrlNavigate}
                placeholder="Enter website URL (e.g., example.com)"
              />
            </div>
          </div>

          {/* Prompt Input */}
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Describe what you want to automate <span className="text-red-400">*</span>
            </label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder={`Example: ${currentConfig.examples[0]}`}
              rows={4}
              className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50 resize-none transition-colors"
              data-testid={selectors.ai.modal.promptInput}
            />
          </div>

          {/* Example Prompts */}
          <div>
            <div className="flex items-center gap-2 mb-2">
              <Lightbulb size={14} className={currentConfig.color} />
              <span className="text-sm text-gray-400">
                Example prompts for {currentConfig.title.toLowerCase()}s
              </span>
            </div>
            <div className="flex flex-wrap gap-2">
              {currentConfig.examples.map((example, index) => (
                <button
                  key={index}
                  onClick={() => handleExampleClick(example)}
                  className="px-3 py-1.5 text-xs text-gray-400 bg-gray-800/50 border border-gray-700 rounded-lg hover:border-gray-600 hover:text-gray-300 transition-colors"
                >
                  {example}
                </button>
              ))}
            </div>
          </div>

          {/* Pro Tip */}
          <div className={`p-3 rounded-xl ${currentConfig.bgColor} border ${currentConfig.borderColor}`}>
            <div className="flex items-start gap-2">
              <Sparkles size={14} className={`${currentConfig.color} mt-0.5 flex-shrink-0`} />
              <p className="text-xs text-gray-300">
                <span className="font-medium">Pro tip:</span> {currentConfig.proTip}
              </p>
            </div>
          </div>

          {/* Validation Message */}
          {!isValid && validationMessage && (
            <div className="px-3 py-2 rounded-xl bg-amber-500/10 border border-amber-500/20">
              <p className="text-sm text-amber-400">{validationMessage}</p>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-between pt-2">
            <div className="text-xs text-gray-500">
              Saving to: <span className="text-gray-400 font-mono">{folder || "root"}/</span>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onClose}
                className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleStart}
                disabled={!isValid}
                data-testid={selectors.ai.modal.generateButton}
                className="px-5 py-2.5 bg-gradient-to-r from-purple-500 to-pink-500 text-white font-medium rounded-xl hover:from-purple-400 hover:to-pink-400 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-purple-500/20 hover:shadow-purple-500/30 flex items-center gap-2"
              >
                <Play size={16} />
                Start AI Recording
              </button>
            </div>
          </div>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default AIPromptModal;
