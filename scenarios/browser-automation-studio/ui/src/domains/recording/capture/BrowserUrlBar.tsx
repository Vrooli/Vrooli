/**
 * BrowserUrlBar Component
 *
 * A browser-like URL input with:
 * - URL normalization (e.g., "reddit.com" → "https://reddit.com")
 * - Dropdown showing normalized URL suggestion
 * - History of visited URLs with fuzzy matching
 * - Security indicator (lock icon for https)
 * - Keyboard navigation (Enter to navigate, Escape to close, arrows to select)
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { addToHistory, loadHistory, saveHistory, scoreHistoryMatch, type HistoryEntry } from './browserUrlHistory';

const MAX_VISIBLE_SUGGESTIONS = 8;

/**
 * Normalize a user input into a proper URL.
 * - Adds https:// if no protocol
 * - Handles common shortcuts
 */
function normalizeUrl(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) return '';

  // Already has a protocol
  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed;
  }

  // localhost with optional port
  if (/^localhost(:\d+)?/i.test(trimmed)) {
    return `http://${trimmed}`;
  }

  // IP address (potentially local)
  if (/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?/.test(trimmed)) {
    // Private IP ranges use http
    const ipMatch = trimmed.match(/^(\d{1,3})\.(\d{1,3})\./);
    const firstOctet = ipMatch?.[1];
    const secondOctet = ipMatch?.[2];
    if (firstOctet && secondOctet) {
      const first = parseInt(firstOctet, 10);
      const second = parseInt(secondOctet, 10);
      if (first === 10 || first === 127 || (first === 192 && second === 168) || (first === 172 && second >= 16 && second <= 31)) {
        return `http://${trimmed}`;
      }
    }
    return `https://${trimmed}`;
  }

  // Looks like a domain (has a dot and valid TLD-like structure)
  if (/^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+/.test(trimmed)) {
    return `https://${trimmed}`;
  }

  // Single word without dots - could be a search or invalid
  // For now, assume it's a domain and add .com
  if (/^[a-zA-Z][a-zA-Z0-9-]*$/.test(trimmed)) {
    return `https://${trimmed}.com`;
  }

  // Fallback: treat as-is with https
  return `https://${trimmed}`;
}

/**
 * Extract display-friendly parts from a URL
 */
function parseUrlForDisplay(url: string): { protocol: string; host: string; path: string; isSecure: boolean } {
  try {
    const parsed = new URL(url);
    return {
      protocol: parsed.protocol,
      host: parsed.host,
      path: parsed.pathname + parsed.search + parsed.hash,
      isSecure: parsed.protocol === 'https:',
    };
  } catch {
    return { protocol: '', host: url, path: '', isSecure: false };
  }
}


interface BrowserUrlBarProps {
  /** Current URL value */
  value: string;
  /** Called when URL changes (may be debounced internally) */
  onChange: (url: string) => void;
  /** Called when user explicitly navigates (Enter key or suggestion click) */
  onNavigate: (url: string) => void;
  /** Placeholder text when empty */
  placeholder?: string;
  /** Whether the bar should be disabled */
  disabled?: boolean;
  /** Current page title (for history tracking) */
  pageTitle?: string;
}

export function BrowserUrlBar({
  value,
  onChange,
  onNavigate,
  placeholder = 'Search or enter URL',
  disabled = false,
  pageTitle,
}: BrowserUrlBarProps) {
  const [inputValue, setInputValue] = useState(value);
  const [isOpen, setIsOpen] = useState(false);
  const [history, setHistory] = useState<HistoryEntry[]>(loadHistory);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [isFocused, setIsFocused] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Sync external value changes
  useEffect(() => {
    setInputValue(value);
  }, [value]);

  // Compute suggestions based on input
  const suggestions = useMemo(() => {
    const trimmed = inputValue.trim();
    const result: Array<{ type: 'normalized' | 'history'; url: string; displayUrl: string; title?: string; score: number }> = [];

    // Add normalized URL suggestion if input looks like a URL
    if (trimmed && !trimmed.startsWith('http')) {
      const normalized = normalizeUrl(trimmed);
      if (normalized && normalized !== trimmed) {
        result.push({
          type: 'normalized',
          url: normalized,
          displayUrl: normalized,
          score: 10000, // Always show first
        });
      }
    }

    // Add matching history items
    if (trimmed) {
      const scored = history
        .map((entry) => ({
          type: 'history' as const,
          url: entry.url,
          displayUrl: entry.url,
          title: entry.title,
          score: scoreHistoryMatch(entry, trimmed),
        }))
        .filter((item) => item.score > 0)
        .sort((a, b) => b.score - a.score)
        .slice(0, MAX_VISIBLE_SUGGESTIONS - 1); // Leave room for normalized

      result.push(...scored);
    } else if (history.length > 0) {
      // Show recent history when input is empty
      const recent = history.slice(0, MAX_VISIBLE_SUGGESTIONS).map((entry) => ({
        type: 'history' as const,
        url: entry.url,
        displayUrl: entry.url,
        title: entry.title,
        score: entry.lastVisited,
      }));
      result.push(...recent);
    }

    return result.slice(0, MAX_VISIBLE_SUGGESTIONS);
  }, [inputValue, history]);

  // Handle click outside to close dropdown
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  // Update history entry with page title when it becomes available
  useEffect(() => {
    if (!value || !pageTitle) return;

    // Find and update the history entry for the current URL
    const currentUrl = value.toLowerCase();
    const existingEntry = history.find((h) => h.url.toLowerCase() === currentUrl);

    if (existingEntry && existingEntry.title !== pageTitle) {
      // Update the title in history
      const updatedHistory = history.map((h) =>
        h.url.toLowerCase() === currentUrl ? { ...h, title: pageTitle } : h
      );
      setHistory(updatedHistory);
      saveHistory(updatedHistory);
    } else if (!existingEntry && pageTitle) {
      // URL not in history yet but we have a title - add it
      const updatedHistory = addToHistory(history, value, pageTitle);
      setHistory(updatedHistory);
      saveHistory(updatedHistory);
    }
  }, [value, pageTitle, history]);

  // Navigate to a URL
  const handleNavigate = useCallback(
    (url: string) => {
      const normalized = normalizeUrl(url);
      setInputValue(normalized);
      setIsOpen(false);
      onChange(normalized);
      onNavigate(normalized);

      // Add to history
      const updatedHistory = addToHistory(history, normalized);
      setHistory(updatedHistory);
      saveHistory(updatedHistory);
    },
    [onChange, onNavigate, history]
  );

  // Handle input changes
  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setInputValue(newValue);
      setSelectedIndex(-1);
      setIsOpen(true);
      // Don't call onChange here - wait for navigation
    },
    []
  );

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, suggestions.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, -1));
          break;
        case 'Enter':
          e.preventDefault();
          if (selectedIndex >= 0 && suggestions[selectedIndex]) {
            handleNavigate(suggestions[selectedIndex].url);
          } else {
            handleNavigate(inputValue);
          }
          inputRef.current?.blur();
          break;
        case 'Escape':
          e.preventDefault();
          setIsOpen(false);
          setInputValue(value);
          inputRef.current?.blur();
          break;
        case 'Tab': {
          // Accept first suggestion on Tab if available
          const firstSuggestion = suggestions[0];
          if (firstSuggestion && selectedIndex === -1) {
            e.preventDefault();
            setInputValue(firstSuggestion.url);
            setSelectedIndex(0);
          }
          break;
        }
      }
    },
    [suggestions, selectedIndex, inputValue, value, handleNavigate]
  );

  // Handle focus
  const handleFocus = useCallback(() => {
    setIsFocused(true);
    setIsOpen(true);
    // Select all text on focus (like a real browser)
    inputRef.current?.select();
  }, []);

  // Handle blur
  const handleBlur = useCallback(() => {
    setIsFocused(false);
    // Delay close to allow click on suggestions
    setTimeout(() => {
      if (!containerRef.current?.contains(document.activeElement)) {
        setIsOpen(false);
      }
    }, 150);
  }, []);

  // Handle suggestion click
  const handleSuggestionClick = useCallback(
    (url: string) => {
      handleNavigate(url);
    },
    [handleNavigate]
  );

  // Clear single history entry
  const handleRemoveHistoryItem = useCallback(
    (e: React.MouseEvent, url: string) => {
      e.stopPropagation();
      e.preventDefault();
      const updated = history.filter((h) => h.url !== url);
      setHistory(updated);
      saveHistory(updated);
    },
    [history]
  );

  // Parse current URL for display
  const parsedUrl = useMemo(() => parseUrlForDisplay(value), [value]);

  return (
    <div className="relative flex-1" ref={containerRef}>
      {/* Main input container - styled like a browser URL bar */}
      <div
        className={`
          flex items-center gap-2 px-3 py-1.5
          bg-gray-100 dark:bg-gray-800
          border border-gray-200 dark:border-gray-700
          rounded-full
          transition-all duration-150
          ${isFocused ? 'bg-white dark:bg-gray-900 ring-2 ring-blue-500 border-transparent shadow-sm' : 'hover:bg-gray-200/50 dark:hover:bg-gray-700/50'}
          ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
        `}
      >
        {/* Security indicator / Site icon */}
        {value && !isFocused ? (
          <div className="flex-shrink-0">
            {parsedUrl.isSecure ? (
              <svg className="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            ) : (
              <svg className="w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
            )}
          </div>
        ) : (
          <svg className="w-4 h-4 text-gray-400 dark:text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        )}

        {/* URL input */}
        <input
          data-testid="browser-url-input"
          ref={inputRef}
          type="text"
          className={`
            flex-1 min-w-0
            bg-transparent
            text-sm text-gray-900 dark:text-gray-100
            placeholder-gray-400 dark:placeholder-gray-500
            focus:outline-none
            ${disabled ? 'cursor-not-allowed' : ''}
          `}
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={handleFocus}
          onBlur={handleBlur}
          placeholder={placeholder}
          disabled={disabled}
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck={false}
        />
      </div>

      {/* Dropdown suggestions */}
      {isOpen && suggestions.length > 0 && (
        <div
          ref={dropdownRef}
          className="absolute left-0 right-0 top-full mt-1 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden z-50"
        >
          {suggestions.map((suggestion, index) => {
            const parsed = parseUrlForDisplay(suggestion.url);
            const isSelected = index === selectedIndex;

            return (
              <div
                key={suggestion.url}
                className={`
                  flex items-center gap-3 px-3 py-2 cursor-pointer
                  transition-colors
                  ${isSelected ? 'bg-blue-50 dark:bg-blue-900/30' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}
                `}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => handleSuggestionClick(suggestion.url)}
                onMouseEnter={() => setSelectedIndex(index)}
              >
                {/* Icon based on type */}
                <div className="flex-shrink-0">
                  {suggestion.type === 'normalized' ? (
                    <svg className="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                    </svg>
                  ) : (
                    <svg className="w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  )}
                </div>

                {/* URL display */}
                <div className="flex-1 min-w-0 overflow-hidden">
                  <div className="flex items-center gap-1 text-sm">
                    {suggestion.type === 'normalized' ? (
                      <>
                        <span className="text-gray-400 dark:text-gray-500">{parsed.protocol}//</span>
                        <span className="text-gray-900 dark:text-gray-100 font-medium truncate">{parsed.host}</span>
                        {parsed.path && parsed.path !== '/' && (
                          <span className="text-gray-500 dark:text-gray-400 truncate">{parsed.path}</span>
                        )}
                      </>
                    ) : (
                      <span className="text-gray-700 dark:text-gray-200 truncate">{suggestion.displayUrl}</span>
                    )}
                  </div>
                  {suggestion.title && (
                    <div className="text-xs text-gray-500 dark:text-gray-400 truncate">{suggestion.title}</div>
                  )}
                </div>

                {/* Type label or remove button for history */}
                {suggestion.type === 'normalized' ? (
                  <span className="flex-shrink-0 text-xs text-blue-500 dark:text-blue-400 font-medium">
                    Go
                  </span>
                ) : (
                  <button
                    type="button"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={(e) => handleRemoveHistoryItem(e, suggestion.url)}
                    className="flex-shrink-0 p-1 text-gray-400 hover:text-red-500 dark:text-gray-500 dark:hover:text-red-400 opacity-0 group-hover:opacity-100 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-all"
                    title="Remove from history"
                    style={{ opacity: isSelected ? 1 : undefined }}
                  >
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                )}
              </div>
            );
          })}

          {/* Keyboard hint */}
          <div className="px-3 py-1.5 bg-gray-50 dark:bg-gray-800/50 border-t border-gray-100 dark:border-gray-700">
            <div className="flex items-center gap-3 text-[10px] text-gray-400 dark:text-gray-500">
              <span><kbd className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-[9px]">↑↓</kbd> Navigate</span>
              <span><kbd className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-[9px]">Enter</kbd> Go</span>
              <span><kbd className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-[9px]">Tab</kbd> Complete</span>
              <span><kbd className="px-1 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-[9px]">Esc</kbd> Cancel</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// useUrlHistory hook moved to useUrlHistory.ts
