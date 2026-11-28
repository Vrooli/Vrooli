/**
 * Settings tab for managing skills.
 *
 * Skills are knowledge modules that get injected into the agent's context
 * to provide methodology and expertise for specific tasks.
 *
 * Features:
 * - View all skills from prompt-manager
 * - Create, edit, delete skills
 * - Search and filter skills
 */

import { useState, useCallback } from "react";
import {
  Pencil,
  Trash2,
  Search,
  RefreshCw,
  Plus,
  Construction,
} from "lucide-react";
import type { SkillWithSource } from "@/lib/types/templates";

interface SkillsSettingsTabProps {
  skills: SkillWithSource[];
  onEditSkill: (skill: SkillWithSource | null) => void; // null for new skill
  onDeleteSkill: (skillId: string) => Promise<void>;
  isLoading?: boolean;
  onSyncSkills?: () => Promise<unknown>; // Callback to sync skills from prompt-manager
  isSyncing?: boolean; // Whether skills are currently being synced
}

export function SkillsSettingsTab({
  skills,
  onEditSkill,
  onDeleteSkill,
  isLoading,
  onSyncSkills,
  isSyncing = false,
}: SkillsSettingsTabProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Filter skills
  const filteredSkills = skills.filter((s) => {
    const matchesSearch =
      !searchQuery ||
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.tags?.some((t) => t.toLowerCase().includes(searchQuery.toLowerCase()));
    return matchesSearch;
  });

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

  const renderSkillItem = (skill: SkillWithSource) => {
    const isDeleting = deletingId === skill.id;

    return (
      <div
        key={skill.id}
        className={`flex items-center justify-between p-2 bg-slate-800/50 border border-white/5 rounded-lg ${
          isDeleting ? "opacity-50" : ""
        }`}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm text-white truncate">{skill.name}</p>
            {skill.draft && (
              <span
                className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-orange-900/50 text-orange-400 border border-orange-500/30"
                title="This skill is a draft and may not be fully working"
              >
                <Construction className="h-2.5 w-2.5" />
                Draft
              </span>
            )}
          </div>
          <p className="text-xs text-slate-500 truncate">
            {skill.modes?.join(" / ") || "No category"}
          </p>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {/* Edit button */}
          <button
            onClick={() => onEditSkill(skill)}
            disabled={isDeleting}
            className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors disabled:opacity-50"
            title="Edit skill"
          >
            <Pencil className="h-4 w-4" />
          </button>

          {/* Delete button */}
          <button
            onClick={() => { void handleDelete(skill.id); }}
            disabled={isDeleting}
            className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors disabled:opacity-50"
            title="Delete skill"
          >
            <Trash2 className="h-4 w-4" />
          </button>
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
            {(isLoading || isSyncing) && (
              <span className="ml-2 text-xs text-slate-500">
                {isSyncing ? "Syncing..." : "Loading..."}
              </span>
            )}
          </h3>
          <div className="flex items-center gap-2">
            {onSyncSkills && (
              <button
                onClick={() => { void onSyncSkills(); }}
                disabled={isSyncing}
                className="flex items-center gap-1 px-2 py-1 text-xs bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors disabled:opacity-50"
                title="Sync skills from prompt-manager"
              >
                <RefreshCw className={`h-3 w-3 ${isSyncing ? "animate-spin" : ""}`} />
                Sync
              </button>
            )}
            <button
              onClick={() => onEditSkill(null)}
              className="flex items-center gap-1 px-2 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors"
            >
              <Plus className="h-3 w-3" />
              New Skill
            </button>
          </div>
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

        {/* Legend - only show draft indicator */}
        <div className="flex items-center gap-4 mb-3 text-xs text-slate-500">
          <span className="flex items-center gap-1 text-orange-400">
            <Construction className="h-3 w-3" />
            Draft
          </span>
        </div>

        {/* Skills - flat list */}
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {filteredSkills.length > 0 ? (
            filteredSkills.map(renderSkillItem)
          ) : (
            <div className="text-sm text-slate-500 text-center py-4">
              {isLoading || isSyncing ? (
                isSyncing ? "Syncing skills..." : "Loading skills..."
              ) : searchQuery ? (
                "No skills found"
              ) : (
                <div className="space-y-2">
                  <p>No skills available</p>
                  {onSyncSkills && (
                    <button
                      onClick={() => { void onSyncSkills(); }}
                      disabled={isSyncing}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors disabled:opacity-50"
                    >
                      <RefreshCw className="h-3.5 w-3.5" />
                      Sync from prompt-manager
                    </button>
                  )}
                </div>
              )}
            </div>
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
          <strong className="text-slate-400">Tip:</strong> All skills are managed in prompt-manager
          and synced automatically. Changes are saved directly to prompt-manager.
        </p>
      </section>
    </div>
  );
}
