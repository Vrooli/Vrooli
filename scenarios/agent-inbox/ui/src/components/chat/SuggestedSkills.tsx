/**
 * SuggestedSkills - Renders AI-suggested skill chips in the message input footer.
 *
 * Uses teal/cyan color scheme to distinguish from attached (amber) skill pills.
 * Each chip shows the skill name with a score badge, and supports click-to-attach
 * and dismiss actions.
 */
import { Loader2, Sparkles, X } from "lucide-react";
import type { SuggestedSkill } from "@/lib/api";

interface SuggestedSkillsProps {
  suggestions: SuggestedSkill[];
  isLoading: boolean;
  onAttach: (skillId: string) => void;
  onDismiss: (skillId: string) => void;
  onDismissAll: () => void;
}

export function SuggestedSkills({
  suggestions,
  isLoading,
  onAttach,
  onDismiss,
  onDismissAll,
}: SuggestedSkillsProps) {
  if (suggestions.length === 0 && !isLoading) {
    return null;
  }

  return (
    <div className="flex items-center gap-1.5 flex-wrap" data-testid="suggested-skills">
      {isLoading && suggestions.length === 0 && (
        <div className="inline-flex items-center gap-1.5 px-2 py-1 text-xs text-teal-400/60">
          <Loader2 className="h-3 w-3 animate-spin" />
          <span>Finding skills...</span>
        </div>
      )}
      {suggestions.length > 0 && (
        <>
          <Sparkles className="h-3 w-3 text-teal-400/60 shrink-0" />
          {suggestions.map((skill) => (
            <button
              key={skill.id}
              onClick={() => onAttach(skill.id)}
              className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-teal-500/15 border border-teal-500/25 text-xs text-teal-400 hover:bg-teal-500/25 hover:border-teal-500/40 transition-colors group"
              title={`${skill.name}\n${skill.description}\nClick to attach`}
              data-testid={`suggested-skill-${skill.id}`}
            >
              <span className="max-w-[120px] truncate">{skill.name}</span>
              {skill.scorePercent > 0 && (
                <span className="text-[10px] text-teal-400/50 tabular-nums">
                  {skill.scorePercent}%
                </span>
              )}
              <span
                role="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onDismiss(skill.id);
                }}
                className="opacity-0 group-hover:opacity-100 hover:text-teal-300 transition-all"
                aria-label={`Dismiss ${skill.name} suggestion`}
              >
                <X className="h-3 w-3" />
              </span>
            </button>
          ))}
          {suggestions.length > 1 && (
            <button
              onClick={onDismissAll}
              className="text-[10px] text-teal-400/40 hover:text-teal-400/70 transition-colors px-1"
              aria-label="Dismiss all suggestions"
            >
              clear
            </button>
          )}
          {isLoading && (
            <Loader2 className="h-3 w-3 animate-spin text-teal-400/40 shrink-0" />
          )}
        </>
      )}
    </div>
  );
}
