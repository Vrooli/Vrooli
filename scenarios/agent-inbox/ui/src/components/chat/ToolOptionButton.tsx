/**
 * ToolOptionButton - Individual tool option in the ToolSelector modal.
 */

import { Wrench, Check } from "lucide-react";
import type { EffectiveTool } from "@/lib/api";

export interface FlatTool {
  scenario: string;
  tool: EffectiveTool;
  index: number;
}

interface ToolOptionButtonProps {
  scenario: string;
  tool: EffectiveTool;
  index: number;
  isSelected: boolean;
  isFocused: boolean;
  onSelect: (scenario: string, toolName: string) => void;
  onFocus: (index: number) => void;
  buttonRef: (el: HTMLButtonElement | null) => void;
}

export function ToolOptionButton({
  scenario,
  tool,
  index,
  isSelected,
  isFocused,
  onSelect,
  onFocus,
  buttonRef,
}: ToolOptionButtonProps) {
  return (
    <button
      key={`${scenario}-${tool.tool.name}`}
      ref={buttonRef}
      onClick={() => onSelect(scenario, tool.tool.name)}
      onFocus={() => onFocus(index)}
      role="option"
      aria-selected={isSelected}
      tabIndex={isFocused ? 0 : -1}
      className={`
        w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-colors
        ${
          isSelected
            ? "bg-violet-500/20 border-violet-500/50"
            : isFocused
              ? "bg-slate-700/50 border-violet-400/50 ring-2 ring-violet-500/30"
              : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
        }
      `}
      data-testid={`tool-option-${tool.tool.name}`}
    >
      <div
        className={`
        flex-shrink-0 p-1.5 rounded-lg
        ${isSelected ? "bg-violet-500/30" : "bg-slate-700"}
      `}
      >
        <Wrench
          className={`h-4 w-4 ${isSelected ? "text-violet-400" : "text-slate-300"}`}
        />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span
            className={`font-medium text-sm ${isSelected ? "text-violet-300" : "text-white"}`}
          >
            {tool.tool.name}
          </span>
          {tool.tool.category && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-400">
              {tool.tool.category}
            </span>
          )}
        </div>
        {tool.tool.description && (
          <p className="text-xs text-slate-400 mt-1 line-clamp-2">
            {tool.tool.description}
          </p>
        )}
      </div>
      {isSelected && (
        <Check className="h-4 w-4 text-violet-400 shrink-0" />
      )}
    </button>
  );
}
