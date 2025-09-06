import { useState } from 'react';
import { X, Sparkles, Wand2, FileCode, Loader } from 'lucide-react';
import { useWorkflowStore } from '../stores/workflowStore';
import toast from 'react-hot-toast';

interface AIPromptModalProps {
  onClose: () => void;
  folder: string;
}

const examplePrompts = [
  "Navigate to example.com, click the login button, fill in username and password, then take a screenshot",
  "Go to a news website, extract all article headlines and their URLs",
  "Search for 'automation' on Google, click the first result, take a full page screenshot",
  "Fill out a contact form with test data and submit it",
  "Monitor a product page for price changes every hour",
];

function AIPromptModal({ onClose, folder }: AIPromptModalProps) {
  const [prompt, setPrompt] = useState('');
  const [workflowName, setWorkflowName] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const { generateWorkflow } = useWorkflowStore();

  const handleGenerate = async () => {
    if (!prompt.trim()) {
      toast.error('Please enter a prompt');
      return;
    }
    
    if (!workflowName.trim()) {
      toast.error('Please enter a workflow name');
      return;
    }

    setIsGenerating(true);
    try {
      await generateWorkflow(prompt, workflowName, folder);
      toast.success('Workflow generated successfully!');
      onClose();
    } catch (error) {
      toast.error('Failed to generate workflow');
    } finally {
      setIsGenerating(false);
    }
  };

  const handleExampleClick = (example: string) => {
    setPrompt(example);
    const name = example.split(',')[0].toLowerCase().replace(/\s+/g, '-').slice(0, 30);
    setWorkflowName(name);
  };

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-flow-node border border-gray-700 rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
              <Sparkles size={20} className="text-white" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">AI Workflow Generator</h2>
              <p className="text-xs text-gray-400">Describe what you want to automate</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="toolbar-button p-2"
          >
            <X size={18} />
          </button>
        </div>
        
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
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
            />
          </div>
          
          <div>
            <div className="flex items-center gap-2 mb-3">
              <Wand2 size={16} className="text-purple-400" />
              <span className="text-sm font-medium text-gray-300">Example Prompts</span>
            </div>
            <div className="space-y-2">
              {examplePrompts.map((example, index) => (
                <button
                  key={index}
                  onClick={() => handleExampleClick(example)}
                  className="w-full text-left p-3 bg-flow-bg rounded-lg border border-gray-700 hover:border-purple-500/50 transition-colors group"
                >
                  <div className="flex items-start gap-2">
                    <FileCode size={14} className="text-gray-500 group-hover:text-purple-400 mt-0.5" />
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
                  <span className="font-medium text-gray-300">Pro tip:</span> Be specific about selectors, wait conditions, and data you want to extract. 
                  The AI will generate a visual workflow you can further customize.
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
      </div>
    </div>
  );
}

export default AIPromptModal;