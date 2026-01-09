import React, { useState } from 'react';

interface QuickStartWidgetProps {
  onAIGenerate: (prompt: string) => void;
  onCreateManual: () => void;
  isGenerating?: boolean;
}

export const QuickStartWidget: React.FC<QuickStartWidgetProps> = ({
  onAIGenerate,
  onCreateManual,
  isGenerating = false
}) => {
  const [prompt, setPrompt] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (prompt.trim() && !isGenerating) {
      onAIGenerate(prompt.trim());
    }
  };

  const examplePrompts = [
    'Log into my dashboard and take a screenshot',
    'Fill out a contact form and submit it',
    'Scrape product prices from an e-commerce site',
    'Check if a website is loading correctly',
  ];

  const handleExampleClick = (example: string) => {
    setPrompt(example);
  };

  return (
    <div className="bg-gradient-to-br from-purple-900/30 via-blue-900/20 to-gray-900 border border-purple-500/30 rounded-lg p-5">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-10 h-10 bg-purple-500/20 rounded-lg">
          <svg className="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <div>
        <h3 className="text-surface font-medium">Quick Start</h3>
          <p className="text-sm text-gray-400">Describe what you want to automate</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="mb-4">
        <div className="relative">
          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Log into example.com and take a screenshot of the dashboard..."
            className="w-full px-4 py-3 pr-24 bg-gray-800/80 border border-gray-600/50 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-purple-500/50 focus:ring-1 focus:ring-purple-500/30 transition-colors"
            disabled={isGenerating}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || isGenerating}
            className="absolute right-2 top-1/2 -translate-y-1/2 px-3 py-1.5 bg-purple-600 hover:bg-purple-500 disabled:bg-gray-600 disabled:cursor-not-allowed text-white text-sm font-medium rounded-md transition-colors flex items-center gap-1.5"
          >
            {isGenerating ? (
              <>
                <div className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                Generate
              </>
            )}
          </button>
        </div>
      </form>

      <div className="flex flex-wrap gap-2 mb-4">
        {examplePrompts.map((example, index) => (
          <button
            key={index}
            onClick={() => handleExampleClick(example)}
            className="px-2.5 py-1 text-xs text-gray-400 bg-gray-800/50 hover:bg-gray-700/50 hover:text-gray-300 border border-gray-700/50 rounded-full transition-colors"
          >
            {example.length > 35 ? example.slice(0, 35) + '...' : example}
          </button>
        ))}
      </div>

      <div className="flex items-center gap-3 pt-3 border-t border-gray-700/50">
        <span className="text-xs text-gray-500">or</span>
        <button
          onClick={onCreateManual}
          className="text-sm text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1.5"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
          Create workflow manually
        </button>
      </div>
    </div>
  );
};
