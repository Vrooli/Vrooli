/**
 * SkillIndicator - Shows when skills are attached for the current message.
 *
 * Displays attached skills as pills in the message input footer area.
 * Uses amber/orange color scheme to distinguish from other indicators.
 */
import { BookOpen, Plus, X } from "lucide-react";
import type { Skill } from "@/lib/types/templates";

interface SkillIndicatorProps {
  skills: Skill[];
  onRemove: (skillId: string) => void;
  onAdd: () => void;
}

export function SkillIndicator({ skills, onRemove, onAdd }: SkillIndicatorProps) {
  if (skills.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-1.5 flex-wrap" data-testid="skill-indicator">
      {skills.map((skill) => (
        <div
          key={skill.id}
          className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-amber-500/20 border border-amber-500/30 text-xs text-amber-400"
          data-testid={`skill-pill-${skill.id}`}
        >
          <BookOpen className="h-3 w-3" />
          <span
            className="max-w-[120px] truncate"
            title={`Skill: ${skill.name}\n${skill.description}`}
          >
            {skill.name}
          </span>
          <button
            onClick={() => onRemove(skill.id)}
            className="hover:text-amber-300 transition-colors"
            aria-label={`Remove ${skill.name} skill`}
            data-testid={`skill-remove-${skill.id}`}
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}
      <button
        onClick={onAdd}
        className="inline-flex items-center gap-1 px-2 py-1 rounded-full border border-dashed border-amber-500/30 text-xs text-amber-400/60 hover:text-amber-400 hover:border-amber-500/50 transition-colors"
        aria-label="Add more skills"
        data-testid="skill-add-button"
      >
        <Plus className="h-3 w-3" />
      </button>
    </div>
  );
}
