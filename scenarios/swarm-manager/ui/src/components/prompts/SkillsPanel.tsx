import type { KeyboardEvent } from "react";
import { selectors } from "../../consts/selectors";
import type { PromptSkillSummary } from "../../types";

const formatUsageLabel = (value: string) => value.replace(/_/g, " ");
const joinParts = (parts?: string[]) => (parts && parts.length > 0 ? parts.join(", ") : "-");

export interface SkillsPanelProps {
  skills: PromptSkillSummary[];
  selectedSkillId: string;
  onSelectSkill: (id: string) => void;
  /** When provided, closing the panel triggers this callback (mobile bottom-sheet). */
  onClose?: () => void;
}

/**
 * Reusable list of prompt skills shown in both the desktop sidebar and
 * the mobile bottom-sheet.
 */
export function SkillsPanel({ skills, selectedSkillId, onSelectSkill, onClose }: SkillsPanelProps) {
  const handleClick = (id: string) => {
    onSelectSkill(id);
    onClose?.();
  };

  return (
    <div className="h-full space-y-3 p-4" data-testid={selectors.prompts.skillsList}>
      <h3 className="text-base font-semibold text-slate-100">Swarm Prompt Skills</h3>
      <div className="max-h-full space-y-2 overflow-auto pr-1">
        {skills.map((skill) => (
          <button
            key={skill.id}
            className={`w-full rounded-md border px-3 py-2 text-left transition ${
              selectedSkillId === skill.id
                ? "border-cyan-500/60 bg-cyan-500/10"
                : "border-slate-700/60 bg-slate-900/30 hover:border-slate-500/60"
            }`}
            onClick={() => handleClick(skill.id)}
          >
            <p className="font-mono text-xs text-cyan-300">{skill.id}</p>
            <p className="mt-1 text-sm text-slate-100">{skill.name}</p>
            <p className="mt-1 text-[11px] uppercase tracking-wide text-slate-500">
              {formatUsageLabel(skill.usage_type)} {"\u2022"} {joinParts(skill.groups)}
            </p>
            <p className="mt-1 text-xs text-slate-400">{skill.impact_summary}</p>
          </button>
        ))}
      </div>
    </div>
  );
}
