import { useState, useRef, useEffect } from "react";
import { Bot, MessageSquare, ChevronDown } from "lucide-react";

export type ChatMode = "llm" | "agent";

interface ModeSelectorProps {
  /** Current selected mode */
  mode: ChatMode;
  /** Called when mode changes */
  onModeChange: (mode: ChatMode) => void;
  /** Whether the selector is disabled */
  disabled?: boolean;
  /** Whether the chat is currently in agent mode (prevents switching) */
  isAgentActive?: boolean;
}

// Define mode options as a tuple to ensure at least one element
const DEFAULT_MODE = {
  value: "llm" as const,
  label: "LLM",
  description: "Chat with AI using LLM completion",
  icon: <MessageSquare className="h-4 w-4" />
};

const MODES: { value: ChatMode; label: string; description: string; icon: React.ReactNode }[] = [
  DEFAULT_MODE,
  {
    value: "agent",
    label: "Agent",
    description: "Agentic coding with tool access",
    icon: <Bot className="h-4 w-4" />
  }
];

/**
 * Mode selector dropdown for choosing between LLM and Agent modes.
 * Attached to the submit button area in MessageInput.
 */
export function ModeSelector({
  mode,
  onModeChange,
  disabled = false,
  isAgentActive = false
}: ModeSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Always returns a valid mode since we have a fallback
  const currentMode = MODES.find(m => m.value === mode) ?? DEFAULT_MODE;

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => !disabled && !isAgentActive && setIsOpen(!isOpen)}
        disabled={disabled || isAgentActive}
        className={`
          flex items-center gap-1.5 px-2 py-1 rounded-md text-sm
          transition-colors duration-150
          ${disabled || isAgentActive
            ? "opacity-50 cursor-not-allowed"
            : "hover:bg-zinc-700 cursor-pointer"
          }
          ${mode === "agent" ? "text-blue-400" : "text-zinc-400"}
        `}
        title={isAgentActive ? "Cannot change mode while agent is running" : `Mode: ${currentMode.label}`}
      >
        {currentMode.icon}
        <span className="hidden sm:inline">{currentMode.label}</span>
        {!isAgentActive && <ChevronDown className="h-3 w-3 opacity-50" />}
      </button>

      {isOpen && (
        <div className="absolute bottom-full left-0 mb-1 w-56 bg-zinc-800 border border-zinc-700 rounded-lg shadow-lg z-50">
          <div className="p-1">
            {MODES.map(option => (
              <button
                key={option.value}
                type="button"
                onClick={() => {
                  onModeChange(option.value);
                  setIsOpen(false);
                }}
                className={`
                  w-full flex items-start gap-3 px-3 py-2 rounded-md text-left
                  transition-colors duration-150
                  ${mode === option.value
                    ? "bg-zinc-700 text-white"
                    : "text-zinc-300 hover:bg-zinc-700/50"
                  }
                `}
              >
                <div className={`mt-0.5 ${mode === option.value ? "text-blue-400" : "text-zinc-400"}`}>
                  {option.icon}
                </div>
                <div>
                  <div className="font-medium">{option.label}</div>
                  <div className="text-xs text-zinc-400">{option.description}</div>
                </div>
              </button>
            ))}
          </div>
          {mode === "agent" && (
            <div className="border-t border-zinc-700 px-3 py-2 text-xs text-zinc-500">
              Agent mode uses claude-code for agentic coding tasks
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default ModeSelector;
