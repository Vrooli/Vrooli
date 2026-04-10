/**
 * Combobox - Dropdown input with filtering and custom value support.
 * Used by CategoryPathEditor for each level of the category path.
 */

import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { X, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ComboboxProps {
  value: string;
  onChange: (value: string) => void;
  suggestions: string[];
  placeholder?: string;
  disabled?: boolean;
  onDelete?: () => void;
  showDelete?: boolean;
  level: number;
}

export function Combobox({
  value,
  onChange,
  suggestions,
  placeholder,
  disabled,
  onDelete,
  showDelete,
  level,
}: ComboboxProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [inputValue, setInputValue] = useState(value);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Sync inputValue with value prop
  useEffect(() => {
    setInputValue(value);
  }, [value]);

  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
      setIsOpen(false);
      // Commit value on blur
      if (inputValue !== value) {
        onChange(inputValue);
      }
    }
  }, [inputValue, value, onChange]);

  const handleEscape = useCallback((event: KeyboardEvent) => {
    if (event.key === "Escape") {
      setIsOpen(false);
      setInputValue(value); // Revert to original
    }
  }, [value]);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("keydown", handleEscape);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [isOpen, handleClickOutside, handleEscape]);

  // Filter suggestions based on input
  const filteredSuggestions = useMemo(() => {
    if (!inputValue.trim()) return suggestions;
    const lower = inputValue.toLowerCase();
    return suggestions.filter((s) => s.toLowerCase().includes(lower));
  }, [suggestions, inputValue]);

  // Check if current value is custom (not in suggestions)
  const isCustomValue = useMemo(() => {
    return inputValue.trim() !== "" && !suggestions.includes(inputValue);
  }, [inputValue, suggestions]);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
    setIsOpen(true);
  }, []);

  const handleSelect = useCallback(
    (suggestion: string) => {
      setInputValue(suggestion);
      onChange(suggestion);
      setIsOpen(false);
    },
    [onChange]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        if (inputValue.trim()) {
          onChange(inputValue.trim());
          setIsOpen(false);
        }
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        setIsOpen(true);
      }
    },
    [inputValue, onChange]
  );

  return (
    <div ref={containerRef} className="relative flex items-center gap-1">
      <div className="relative flex-1">
        <div className="relative">
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={handleInputChange}
            onFocus={() => setIsOpen(true)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder || `Level ${level + 1}`}
            disabled={disabled}
            className={cn(
              "w-full px-3 py-2 pr-8 bg-slate-800 border border-white/10 rounded-lg text-white text-sm",
              "placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500",
              disabled && "opacity-50 cursor-not-allowed"
            )}
          />
          <button
            type="button"
            onClick={() => !disabled && setIsOpen(!isOpen)}
            disabled={disabled}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
          >
            <ChevronDown
              className={cn("h-4 w-4 transition-transform", isOpen && "rotate-180")}
            />
          </button>
        </div>

        {/* Custom value badge */}
        {isCustomValue && inputValue.trim() && (
          <span className="absolute -right-14 top-1/2 -translate-y-1/2 px-1.5 py-0.5 text-[10px] font-medium bg-emerald-600/30 text-emerald-300 rounded">
            NEW
          </span>
        )}

        {/* Dropdown */}
        {isOpen && (filteredSuggestions.length > 0 || inputValue.trim()) && (
          <div
            className={cn(
              "absolute z-50 mt-1 w-full max-h-48 overflow-y-auto",
              "bg-slate-900 border border-white/10 rounded-lg shadow-xl",
              "animate-in fade-in-0 zoom-in-95 duration-100"
            )}
          >
            {filteredSuggestions.length > 0 ? (
              filteredSuggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  onClick={() => handleSelect(suggestion)}
                  className={cn(
                    "w-full px-3 py-2 text-left text-sm transition-colors",
                    "hover:bg-white/5 focus:bg-white/5 focus:outline-none",
                    suggestion === value ? "text-indigo-300 bg-indigo-600/20" : "text-slate-300"
                  )}
                >
                  {suggestion}
                </button>
              ))
            ) : (
              <div className="px-3 py-2 text-sm text-slate-500">
                Press Enter to create &quot;{inputValue}&quot;
              </div>
            )}
          </div>
        )}
      </div>

      {/* Delete button */}
      {showDelete && (
        <button
          type="button"
          onClick={onDelete}
          disabled={disabled}
          className={cn(
            "p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors",
            disabled && "opacity-50 cursor-not-allowed"
          )}
          title="Remove level"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
