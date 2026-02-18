import { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { Bot, MessageSquare, ChevronDown, Settings, Link2 } from "lucide-react";

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
  /** Open settings focused on agent tab */
  onOpenAgentSettings?: () => void;
  /** Open the attach run modal */
  onOpenAttachModal?: () => void;
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
 *
 * Used in both EmptyState (homepage) and ChatView (active chat).
 * The dropdown is rendered via a portal so it isn't clipped by
 * ancestor overflow containers.
 */
export function ModeSelector({
  mode,
  onModeChange,
  disabled = false,
  isAgentActive = false,
  onOpenAgentSettings,
  onOpenAttachModal,
}: ModeSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [dropdownPos, setDropdownPos] = useState<{ top: number; left: number } | null>(null);

  // Position the portal dropdown relative to the toggle button, clamped to viewport
  useEffect(() => {
    if (!isOpen || !toggleRef.current) {
      setDropdownPos(null);
      return;
    }
    const rect = toggleRef.current.getBoundingClientRect();
    const dropdownWidth = 256; // w-64 = 16rem = 256px
    const dropdownHeight = 120; // approximate height of the dropdown
    const margin = 8;

    // Default: open upward above the button
    let top = rect.top - margin;
    let left = rect.left;

    // If it would go above viewport, open below the button instead
    if (top - dropdownHeight < margin) {
      top = rect.bottom + margin;
    }

    // Clamp horizontally to viewport
    if (left + dropdownWidth > window.innerWidth - margin) {
      left = window.innerWidth - dropdownWidth - margin;
    }
    if (left < margin) {
      left = margin;
    }

    setDropdownPos({ top, left });
  }, [isOpen]);

  // Close dropdown when clicking outside both the toggle and the portal dropdown
  useEffect(() => {
    if (!isOpen) return;

    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node;
      if (
        toggleRef.current?.contains(target) ||
        dropdownRef.current?.contains(target)
      ) {
        return;
      }
      setIsOpen(false);
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen]);

  // Close the dropdown first, then change mode. This prevents a React reconciliation
  // crash where the dropdown div and newly-appearing action buttons occupy the same
  // child positions, causing "removeChild" errors during DOM patching.
  const handleSelectMode = useCallback((newMode: ChatMode) => {
    setIsOpen(false);
    onModeChange(newMode);
  }, [onModeChange]);

  const currentMode = MODES.find((m) => m.value === mode) ?? DEFAULT_MODE;

  const showAttach = mode === "agent" && !isAgentActive && !!onOpenAttachModal;
  const showSettings = mode === "agent" && !!onOpenAgentSettings;

  // Determine if dropdown opens upward or downward based on position
  const opensUpward = dropdownPos
    ? dropdownPos.top < (toggleRef.current?.getBoundingClientRect().bottom ?? 0)
    : true;

  return (
    <div className="relative flex items-center gap-1">
      <button
        ref={toggleRef}
        type="button"
        onClick={() => !disabled && !isAgentActive && setIsOpen(!isOpen)}
        disabled={disabled || isAgentActive}
        className={`
          flex items-center gap-1.5 px-2 py-1 rounded-md text-sm
          transition-colors duration-150
          ${disabled || isAgentActive
            ? "opacity-50 cursor-not-allowed"
            : "hover:bg-white/10 cursor-pointer"
          }
          ${mode === "agent" ? "text-blue-400" : "text-slate-400"}
        `}
        title={isAgentActive ? "Cannot change mode while agent is running" : `Mode: ${currentMode.label}`}
      >
        {currentMode.icon}
        <span className="hidden sm:inline">{currentMode.label}</span>
        {!isAgentActive && <ChevronDown className="h-3 w-3 opacity-50" />}
      </button>

      {/* Action buttons — stable wrapper so they don't interfere with the dropdown. */}
      {(showAttach || showSettings) && (
        <div className="flex items-center gap-1">
          {showAttach && (
            <button
              type="button"
              onClick={() => {
                setIsOpen(false);
                onOpenAttachModal!();
              }}
              className="p-1.5 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
              title="Attach to existing run"
              aria-label="Attach to existing run"
            >
              <Link2 className="h-3.5 w-3.5" />
            </button>
          )}
          {showSettings && (
            <button
              type="button"
              onClick={() => {
                setIsOpen(false);
                onOpenAgentSettings!();
              }}
              className="p-1.5 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
              title="Open Agent settings"
              aria-label="Open Agent settings"
            >
              <Settings className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      )}

      {/* Portal the dropdown so it escapes overflow-hidden/auto ancestors */}
      {isOpen && dropdownPos && createPortal(
        <div
          ref={dropdownRef}
          className="fixed w-64 bg-slate-900 border border-white/10 rounded-lg shadow-xl z-[9999] animate-in fade-in-0 zoom-in-95 duration-100"
          style={{
            top: dropdownPos.top,
            left: dropdownPos.left,
            transform: opensUpward ? "translateY(-100%)" : undefined,
          }}
        >
          <div className="p-1">
            {MODES.map((option) => (
              <div
                key={option.value}
                className={`
                  flex items-center gap-1 rounded-md
                  ${mode === option.value ? "bg-white/10" : "hover:bg-white/5"}
                `}
              >
                <button
                  type="button"
                  onClick={() => handleSelectMode(option.value)}
                  className="flex-1 flex items-start gap-3 px-3 py-2 text-left transition-colors duration-150"
                >
                  <div className={`mt-0.5 ${mode === option.value ? "text-blue-400" : "text-slate-400"}`}>
                    {option.icon}
                  </div>
                  <div>
                    <div className={`font-medium ${mode === option.value ? "text-white" : "text-slate-300"}`}>
                      {option.label}
                    </div>
                    <div className="text-xs text-slate-400">{option.description}</div>
                  </div>
                </button>

                {option.value === "agent" && onOpenAgentSettings && (
                  <button
                    type="button"
                    onClick={(event) => {
                      event.stopPropagation();
                      setIsOpen(false);
                      onOpenAgentSettings();
                    }}
                    className="mr-1 p-1.5 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
                    title="Open Agent settings"
                    aria-label="Open Agent settings"
                  >
                    <Settings className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            ))}
          </div>
          {mode === "agent" && (
            <div className="border-t border-white/10 px-3 py-2 text-xs text-slate-500">
              Agent mode uses claude-code for agentic coding tasks
            </div>
          )}
        </div>,
        document.body
      )}
    </div>
  );
}

export default ModeSelector;
