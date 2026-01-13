/**
 * Settings tab for managing skills.
 *
 * Skills are knowledge modules that get injected into the agent's context
 * to provide methodology and expertise for specific tasks.
 *
 * Features:
 * - View all skills (default and custom)
 * - Create, edit, delete skills
 * - Reset modified defaults to original
 * - Search and filter skills
 */

import { useState, useCallback } from "react";
import {
  Pencil,
  Trash2,
  RotateCcw,
  Search,
  FileText,
  User,
  RefreshCw,
  Plus,
} from "lucide-react";
import type { SkillWithSource } from "@/lib/types/templates";

interface SkillsSettingsTabProps {
  skills: SkillWithSource[];
  onEditSkill: (skill: SkillWithSource | null) => void; // null for new skill
  onDeleteSkill: (skillId: string) => Promise<void>;
  onResetSkill: (skillId: string) => Promise<void>;
  isLoading?: boolean;
}

function getSourceBadge(source: SkillWithSource["source"]) {
  switch (source) {
    case "default":
      return {
        label: "Default",
        className: "bg-slate-700 text-slate-400",
        icon: FileText,
      };
    case "user":
      return {
        label: "Custom",
        className: "bg-indigo-900/50 text-indigo-400",
        icon: User,
      };
    case "modified":
      return {
        label: "Modified",
        className: "bg-amber-900/50 text-amber-400",
        icon: RefreshCw,
      };
  }
}

export function SkillsSettingsTab({
  skills,
  onEditSkill,
  onDeleteSkill,
  onResetSkill,
  isLoading,
}: SkillsSettingsTabProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [resettingId, setResettingId] = useState<string | null>(null);

  // Filter skills
  const filteredSkills = skills.filter((s) => {
    const matchesSearch =
      !searchQuery ||
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.tags?.some((t) => t.toLowerCase().includes(searchQuery.toLowerCase()));
    return matchesSearch;
  });

  // Group skills by source for better organization
  const defaultSkills = filteredSkills.filter((s) => s.source === "default");
  const modifiedSkills = filteredSkills.filter((s) => s.source === "modified");
  const userSkills = filteredSkills.filter((s) => s.source === "user");

  const handleDelete = useCallback(
    async (skillId: string) => {
      setDeletingId(skillId);
      try {
        await onDeleteSkill(skillId);
      } finally {
        setDeletingId(null);
      }
    },
    [onDeleteSkill]
  );

  const handleReset = useCallback(
    async (skillId: string) => {
      setResettingId(skillId);
      try {
        await onResetSkill(skillId);
      } finally {
        setResettingId(null);
      }
    },
    [onResetSkill]
  );

  const renderSkillItem = (skill: SkillWithSource) => {
    const badge = getSourceBadge(skill.source);
    const BadgeIcon = badge.icon;
    const isDeleting = deletingId === skill.id;
    const isResetting = resettingId === skill.id;
    const isOperating = isDeleting || isResetting;

    return (
      <div
        key={skill.id}
        className={`flex items-center justify-between p-2 bg-slate-800/50 border border-white/5 rounded-lg ${
          isOperating ? "opacity-50" : ""
        }`}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm text-white truncate">{skill.name}</p>
            <span
              className={`flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded ${badge.className}`}
            >
              <BadgeIcon className="h-2.5 w-2.5" />
              {badge.label}
            </span>
          </div>
          <p className="text-xs text-slate-500 truncate">
            {skill.modes?.join(" / ") || skill.category || "No category"}
          </p>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {/* Edit button - available for ALL skills */}
          <button
            onClick={() => onEditSkill(skill)}
            disabled={isOperating}
            className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors disabled:opacity-50"
            title="Edit skill"
          >
            <Pencil className="h-4 w-4" />
          </button>

          {/* Reset button - only for modified skills */}
          {skill.source === "modified" && (
            <button
              onClick={() => handleReset(skill.id)}
              disabled={isOperating}
              className="p-1.5 rounded hover:bg-white/10 text-amber-400 hover:text-amber-300 transition-colors disabled:opacity-50"
              title="Reset to default"
            >
              <RotateCcw className={`h-4 w-4 ${isResetting ? "animate-spin" : ""}`} />
            </button>
          )}

          {/* Delete button - only for user and modified skills */}
          {(skill.source === "user" || skill.source === "modified") && (
            <button
              onClick={() => handleDelete(skill.id)}
              disabled={isOperating}
              className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors disabled:opacity-50"
              title={skill.source === "modified" ? "Remove customization" : "Delete skill"}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Skills List */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-slate-300">
            Skills ({skills.length})
            {isLoading && (
              <span className="ml-2 text-xs text-slate-500">Loading...</span>
            )}
          </h3>
          <button
            onClick={() => onEditSkill(null)}
            className="flex items-center gap-1 px-2 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors"
          >
            <Plus className="h-3 w-3" />
            New Skill
          </button>
        </div>

        {/* Search */}
        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search skills..."
            className="w-full pl-9 pr-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>

        {/* Legend */}
        <div className="flex items-center gap-4 mb-3 text-xs text-slate-500">
          <span className="flex items-center gap-1">
            <FileText className="h-3 w-3" />
            Default
          </span>
          <span className="flex items-center gap-1 text-amber-400">
            <RefreshCw className="h-3 w-3" />
            Modified
          </span>
          <span className="flex items-center gap-1 text-indigo-400">
            <User className="h-3 w-3" />
            Custom
          </span>
        </div>

        {/* Skills */}
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {/* User skills first */}
          {userSkills.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium">
                Custom Skills
              </p>
              {userSkills.map(renderSkillItem)}
            </div>
          )}

          {/* Modified skills */}
          {modifiedSkills.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium mt-3">
                Modified Skills
              </p>
              {modifiedSkills.map(renderSkillItem)}
            </div>
          )}

          {/* Default skills */}
          {defaultSkills.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium mt-3">
                Default Skills
              </p>
              {defaultSkills.map(renderSkillItem)}
            </div>
          )}

          {filteredSkills.length === 0 && (
            <p className="text-sm text-slate-500 text-center py-4">
              {isLoading ? "Loading skills..." : "No skills found"}
            </p>
          )}
        </div>
      </section>

      {/* Info */}
      <section className="text-xs text-slate-500 space-y-1">
        <p>
          <strong className="text-slate-400">About Skills:</strong> Skills are knowledge modules
          that get injected into the agent's context to provide methodology and expertise.
        </p>
        <p>
          <strong className="text-slate-400">Tip:</strong> All skills are editable.
          Editing a default skill creates your own customized version.
        </p>
        <p>
          Use the <RotateCcw className="h-3 w-3 inline text-amber-400" /> button to
          reset a modified skill back to its original default.
        </p>
      </section>
    </div>
  );
}
