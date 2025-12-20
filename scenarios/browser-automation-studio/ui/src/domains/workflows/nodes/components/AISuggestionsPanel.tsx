/**
 * AISuggestionsPanel Component
 *
 * Panel for AI-powered element suggestions. Users describe what they
 * want to click and the AI proposes matching selectors.
 */

import { FC, memo, useCallback, useState } from 'react';
import { Brain, Loader2, Wand2 } from 'lucide-react';
import toast from 'react-hot-toast';
import type { ElementInfo } from '@/types/elements';
import { getConfig } from '@/config';
import { logger } from '@utils/logger';

export interface AISuggestionsPanelProps {
  /** Node ID for logging */
  nodeId: string;
  /** The effective URL to analyze */
  effectiveUrl: string | null;
  /** Upstream URL for logging fallback */
  upstreamUrl?: string | null;
  /** Current suggestions */
  suggestions: ElementInfo[];
  /** Update suggestions list */
  onSuggestionsChange: (suggestions: ElementInfo[]) => void;
  /** Called when a suggestion is selected */
  onSelectSuggestion: (selector: string, elementInfo: ElementInfo) => void;
  /** Called when hovering over a suggestion */
  onHoverSuggestion: (suggestion: ElementInfo | null) => void;
  /** Close the panel */
  onClose: () => void;
}

const AISuggestionsPanel: FC<AISuggestionsPanelProps> = ({
  nodeId,
  effectiveUrl,
  upstreamUrl,
  suggestions,
  onSuggestionsChange,
  onSelectSuggestion,
  onHoverSuggestion,
  onClose,
}) => {
  const [intent, setIntent] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const runSuggestions = useCallback(async () => {
    if (!effectiveUrl) {
      toast.error('Set a page URL before requesting AI suggestions');
      return;
    }

    const trimmedIntent = intent.trim();
    if (!trimmedIntent) {
      toast.error('Describe the element you want to click first');
      return;
    }

    setIsLoading(true);
    onHoverSuggestion(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/ai-analyze-elements`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: effectiveUrl, intent: trimmedIntent }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Failed to analyze page');
      }

      const result: ElementInfo[] = await response.json();
      const normalized = Array.isArray(result) ? result : [];
      onSuggestionsChange(normalized);

      if (normalized.length === 0) {
        toast('AI could not find matching elements. Try refining the description.', {
          icon: '🤔',
        });
      } else {
        toast.success(`Found ${normalized.length} suggestion${normalized.length > 1 ? 's' : ''}`);
      }
    } catch (error) {
      logger.error(
        'Failed to fetch AI suggestions',
        {
          component: 'AISuggestionsPanel',
          action: 'runSuggestions',
          nodeId,
          url: effectiveUrl ?? upstreamUrl ?? null,
        },
        error,
      );
      const message = error instanceof Error ? error.message : 'AI suggestions failed';
      toast.error(message);
    } finally {
      setIsLoading(false);
    }
  }, [effectiveUrl, intent, nodeId, onHoverSuggestion, onSuggestionsChange, upstreamUrl]);

  const handleSelectSuggestion = useCallback(
    (suggestion: ElementInfo) => {
      const primarySelector = suggestion.selectors?.[0]?.selector ?? '';
      if (!primarySelector) {
        toast.error('Suggestion does not include a usable selector');
        return;
      }
      onSelectSuggestion(primarySelector, suggestion);
      onClose();
    },
    [onClose, onSelectSuggestion],
  );

  return (
    <div className="border border-gray-800 rounded-lg bg-flow-bg/60 p-3 space-y-3">
      <div className="flex items-center gap-2 text-xs text-gray-300">
        <Wand2 size={14} className="text-purple-400" />
        <span>Describe what you want to click and let AI propose selectors.</span>
      </div>

      <div className="space-y-2">
        <label className="flex items-center justify-between text-[11px] text-gray-400">
          <span>Description</span>
          <span className="text-gray-600">example: "primary login button"</span>
        </label>
        <textarea
          rows={2}
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          placeholder="What element should be clicked?"
          value={intent}
          onChange={(event) => setIntent(event.target.value)}
        />
        <div className="flex items-center justify-end">
          <button
            type="button"
            onClick={runSuggestions}
            disabled={isLoading}
            className="inline-flex items-center gap-1 px-3 py-1.5 text-xs rounded-md bg-purple-600 text-white hover:bg-purple-500 transition-colors disabled:opacity-60"
          >
            {isLoading ? (
              <>
                <Loader2 size={12} className="animate-spin" />
                Analyzing…
              </>
            ) : (
              <>
                <Brain size={12} />
                Suggest selectors
              </>
            )}
          </button>
        </div>
      </div>

      {suggestions.length > 0 && (
        <div className="space-y-2">
          <p className="text-[11px] text-gray-400">AI suggestions</p>
          <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
            {suggestions.map((suggestion, index) => {
              const primarySelector = suggestion.selectors?.[0]?.selector ?? '';
              const tagName = suggestion.tagName ? suggestion.tagName.toLowerCase() : 'element';
              const confidence = suggestion.confidence ?? 0;

              return (
                <button
                  key={`${primarySelector}-${index}`}
                  type="button"
                  className="w-full text-left border border-gray-700 rounded-md bg-flow-bg/80 hover:border-flow-accent hover:bg-flow-bg/60 transition-colors px-3 py-2 text-xs"
                  onMouseEnter={() => onHoverSuggestion(suggestion)}
                  onMouseLeave={() => onHoverSuggestion(null)}
                  onClick={() => handleSelectSuggestion(suggestion)}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="font-semibold text-gray-100 truncate">
                      {primarySelector || '(no selector)'}
                    </div>
                    <div className="text-[10px] text-gray-400 whitespace-nowrap">
                      {confidence ? `${Math.round(confidence * 100)}% confidence` : 'confidence n/a'}
                    </div>
                  </div>
                  <div className="mt-1 text-[11px] text-gray-400 flex flex-wrap gap-2">
                    <span className="px-1.5 py-0.5 rounded bg-gray-800 text-gray-200">{tagName}</span>
                    {suggestion.text && (
                      <span className="truncate">
                        text: "{suggestion.text.slice(0, 40)}
                        {suggestion.text.length > 40 ? '…' : ''}"
                      </span>
                    )}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

export default memo(AISuggestionsPanel);
