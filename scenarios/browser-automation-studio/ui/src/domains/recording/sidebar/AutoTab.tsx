/**
 * AutoTab Component
 *
 * The Auto (AI) tab content for the unified sidebar.
 * Provides a chat interface for AI navigation.
 *
 * Features:
 * - Message list with scroll-to-bottom behavior
 * - Input field for prompts
 * - Settings button to open AI settings modal
 * - Message bubbles with status, abort, and human intervention
 */

import { useState, useRef, useEffect, useCallback } from 'react';
import { AIMessageBubble } from './AIMessageBubble';
import { AISettingsModal } from './AISettingsModal';
import type { AIMessage, AISettings } from './types';
import { STORAGE_KEYS } from './types';
import type { VisionModelSpec } from '../ai-navigation/types';

// Example prompts to inspire users
const EXAMPLE_PROMPTS = [
  'Search for "weather forecast" and tell me the temperature',
  'Log in to my account using the credentials saved in the form',
  'Find the contact page and fill out the inquiry form',
  'Navigate to the products section and add the first item to cart',
];

// ============================================================================
// Types
// ============================================================================

export interface AutoTabProps {
  /** Messages in the conversation */
  messages: AIMessage[];
  /** Whether AI navigation is in progress */
  isNavigating: boolean;
  /** Current AI settings */
  settings: AISettings;
  /** Available vision models */
  availableModels: VisionModelSpec[];
  /** Send a new message */
  onSendMessage: (prompt: string) => void;
  /** Abort current navigation */
  onAbort: () => void;
  /** Resume after human intervention */
  onHumanDone: () => void;
  /** Update AI settings */
  onSettingsChange: (settings: AISettings) => void;
  /** Clear conversation */
  onClear?: () => void;
  /** Optional className for the container */
  className?: string;
}

// ============================================================================
// Component
// ============================================================================

export function AutoTab({
  messages,
  isNavigating,
  settings,
  availableModels,
  onSendMessage,
  onAbort,
  onHumanDone,
  onSettingsChange,
  onClear,
  className = '',
}: AutoTabProps) {
  // Load initial value from localStorage
  const [inputValue, setInputValue] = useState(() => {
    if (typeof window === 'undefined') return '';
    return localStorage.getItem(STORAGE_KEYS.AI_INPUT_DRAFT) || '';
  });
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Persist input to localStorage when it changes
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (inputValue) {
      localStorage.setItem(STORAGE_KEYS.AI_INPUT_DRAFT, inputValue);
    } else {
      localStorage.removeItem(STORAGE_KEYS.AI_INPUT_DRAFT);
    }
  }, [inputValue]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const handleSubmit = useCallback(
    (e?: React.FormEvent) => {
      e?.preventDefault();
      if (!inputValue.trim() || isNavigating) return;

      onSendMessage(inputValue.trim());
      setInputValue('');
      // Clear from localStorage when sent
      if (typeof window !== 'undefined') {
        localStorage.removeItem(STORAGE_KEYS.AI_INPUT_DRAFT);
      }
    },
    [inputValue, isNavigating, onSendMessage]
  );

  // Handle clicking an example prompt
  const handleExampleClick = useCallback((example: string) => {
    setInputValue(example);
    inputRef.current?.focus();
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit]
  );

  const handleSettingsSave = useCallback(
    (newSettings: AISettings) => {
      onSettingsChange(newSettings);
      setIsSettingsOpen(false);
    },
    [onSettingsChange]
  );

  return (
    <div className={`h-full flex flex-col ${className}`}>
      {/* Header: Settings button and clear */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            AI Navigator
          </span>
        </div>
        <div className="flex items-center gap-1">
          {onClear && messages.length > 0 && (
            <button
              onClick={onClear}
              className="p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-md transition-colors"
              title="Clear conversation"
              aria-label="Clear conversation"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          )}
          <button
            onClick={() => setIsSettingsOpen(true)}
            className="p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-md transition-colors"
            title="AI Settings"
            aria-label="AI Settings"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </button>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto px-3 py-2 space-y-3">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-center text-gray-500 dark:text-gray-400 px-4">
            <svg
              className="w-12 h-12 mb-3 text-gray-300 dark:text-gray-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
            <p className="text-sm font-medium mb-1">AI Navigator</p>
            <p className="text-xs mb-4">
              Describe where you want to navigate and the AI will guide the browser there.
            </p>
            {/* Example prompts */}
            <div className="w-full max-w-xs space-y-2">
              <p className="text-xs font-medium text-gray-400 dark:text-gray-500 mb-2">Try something like:</p>
              {EXAMPLE_PROMPTS.map((example, index) => (
                <button
                  key={index}
                  onClick={() => handleExampleClick(example)}
                  className="w-full text-left px-3 py-2 text-xs text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                >
                  "{example}"
                </button>
              ))}
            </div>
          </div>
        ) : (
          messages.map((message) => (
            <AIMessageBubble
              key={message.id}
              message={message}
              onAbort={message.canAbort ? onAbort : undefined}
              onHumanDone={message.status === 'awaiting_human' ? onHumanDone : undefined}
            />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
        <form onSubmit={handleSubmit} className="flex gap-2">
          <textarea
            ref={inputRef}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={isNavigating ? 'Navigation in progress...' : 'Where do you want to go?'}
            disabled={isNavigating}
            className="flex-1 min-h-[40px] max-h-[120px] px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed"
            rows={1}
            aria-label="Navigation prompt"
          />
          <button
            type="submit"
            disabled={!inputValue.trim() || isNavigating}
            className="px-4 py-2 text-sm font-medium text-white bg-purple-500 rounded-lg hover:bg-purple-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            aria-label="Send message"
          >
            {isNavigating ? (
              <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
                />
              </svg>
            )}
          </button>
        </form>
      </div>

      {/* Settings Modal */}
      <AISettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        currentSettings={settings}
        onSaveSettings={handleSettingsSave}
        availableModels={availableModels}
      />
    </div>
  );
}
