/**
 * SkillSelectorItem - Individual skill row in the SkillSelector modal.
 *
 * Shows preview button, skill info (icon, name, tags, description), and selection checkbox.
 */
import { Eye, Check, Construction, BookOpen } from "lucide-react";
import * as LucideIcons from "lucide-react";
import type { ComponentType, SVGProps } from "react";
import type { Skill } from "@/lib/types/templates";

type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

interface SkillSelectorItemProps {
  skill: Skill;
  isSelected: boolean;
  isFocused: boolean;
  isSkillFocused: boolean;
  isPreviewFocused: boolean;
  skillIndex: number;
  onToggle: (skillId: string) => void;
  onPreview: (skill: Skill) => void;
  onFocus: (index: number) => void;
  skillRef: (el: HTMLButtonElement | null) => void;
  previewRef: (el: HTMLButtonElement | null) => void;
}

export function SkillSelectorItem({
  skill,
  isSelected,
  isFocused,
  isSkillFocused,
  isPreviewFocused,
  skillIndex,
  onToggle,
  onPreview,
  onFocus,
  skillRef,
  previewRef,
}: SkillSelectorItemProps) {
  const IconComponent = getIconComponent(skill.icon || "BookOpen");

  return (
    <div
      className={`
        flex items-start gap-3 p-3 rounded-lg border transition-colors
        ${
          isSelected
            ? "bg-amber-500/20 border-amber-500/50"
            : isFocused
              ? "bg-slate-700/50 border-amber-400/50"
              : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
        }
      `}
    >
      {/* Preview button */}
      <button
        ref={previewRef}
        onClick={() => onPreview(skill)}
        tabIndex={isPreviewFocused ? 0 : -1}
        className={`
          flex-shrink-0 p-1.5 rounded-lg transition-colors
          ${isPreviewFocused
            ? "bg-indigo-500/30 ring-2 ring-indigo-500/50"
            : "bg-slate-700 hover:bg-slate-600"
          }
        `}
        title="Preview skill"
      >
        <Eye className={`h-4 w-4 ${isPreviewFocused ? "text-indigo-300" : "text-slate-400"}`} />
      </button>

      {/* Main skill button */}
      <button
        ref={skillRef}
        onClick={() => onToggle(skill.id)}
        onFocus={() => onFocus(skillIndex)}
        role="option"
        aria-selected={isSelected}
        tabIndex={isSkillFocused ? 0 : -1}
        className="flex-1 text-left min-w-0"
      >
        <div className="flex items-start gap-3">
          <div className={`flex-shrink-0 p-1.5 rounded-lg ${isSelected ? "bg-amber-500/30" : "bg-slate-700"}`}>
            <IconComponent className={`h-4 w-4 ${isSelected ? "text-amber-400" : "text-slate-300"}`} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className={`font-medium text-sm ${isSelected ? "text-amber-300" : "text-white"}`}>
                {skill.name}
              </span>
              {skill.draft && (
                <span
                  className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-orange-900/50 text-orange-400 border border-orange-500/30"
                  title="This skill is a draft and may not be fully working"
                >
                  <Construction className="h-2.5 w-2.5" />
                  Draft
                </span>
              )}
              {skill.tags && skill.tags.length > 0 && (
                <div className="flex gap-1">
                  {skill.tags.slice(0, 2).map((tag) => (
                    <span
                      key={tag}
                      className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-400"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </div>
            <p className="text-xs text-slate-400 mt-1 line-clamp-2">
              {skill.description}
            </p>
          </div>
        </div>
      </button>

      {/* Selection checkbox */}
      <button
        type="button"
        onClick={() => onToggle(skill.id)}
        tabIndex={-1}
        className={`
          flex-shrink-0 w-5 h-5 rounded border flex items-center justify-center transition-colors
          ${isSelected ? "bg-amber-500 border-amber-500" : "border-white/20 hover:border-amber-400/50"}
        `}
        aria-label={isSelected ? "Deselect skill" : "Select skill"}
      >
        {isSelected && <Check className="h-3 w-3 text-white" />}
      </button>
    </div>
  );
}
