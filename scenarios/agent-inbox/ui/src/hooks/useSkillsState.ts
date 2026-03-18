/**
 * useSkillsState - Skills loading, selection, sync, and slash command building.
 *
 * Extracted from useTemplatesAndSkills.ts for modularity.
 */

import { useCallback, useEffect, useState } from "react";
import { getAllSkills, invalidateSkillsCache, syncSkills as dataSyncSkills } from "@/data/skills";
import type {
  Skill,
  SkillPayload,
  SlashCommand,
  SlashCommandType,
  TemplateWithSource,
} from "@/lib/types/templates";
import type { SkillResponse } from "@/lib/api";

export interface UseSkillsStateReturn {
  // Data
  skills: SkillResponse[];
  skillsLoading: boolean;

  // Skills sync and refresh
  refreshSkills: () => Promise<void>;
  syncSkills: () => Promise<{ success: boolean; skillCount: number; error?: string }>;

  // Skills selection state
  selectedSkillIds: string[];
  addSkill: (skillId: string) => void;
  removeSkill: (skillId: string) => void;
  toggleSkill: (skillId: string) => void;
  clearSkills: () => void;
  addSuggestedSkills: (skillIds: string[]) => void;
  getSelectedSkills: () => Skill[];
  buildSkillPayloads: (skillIds: string[]) => SkillPayload[];

  // Slash commands (needs templates for building full list)
  getAllCommands: (templates: TemplateWithSource[]) => SlashCommand[];
  filterCommands: (query: string, templates: TemplateWithSource[]) => SlashCommand[];
}

export function useSkillsState(): UseSkillsStateReturn {
  const [skills, setSkills] = useState<SkillResponse[]>([]);
  const [skillsLoading, setSkillsLoading] = useState(true);
  const [selectedSkillIds, setSelectedSkillIds] = useState<string[]>([]);

  // Load skills on mount
  useEffect(() => {
    let mounted = true;

    async function loadSkills() {
      setSkillsLoading(true);
      try {
        const loaded = await getAllSkills();
        if (mounted) setSkills(loaded);
      } catch (err) {
        console.error("Failed to load skills:", err);
      } finally {
        if (mounted) setSkillsLoading(false);
      }
    }

    loadSkills();
    return () => { mounted = false; };
  }, []);

  const refreshSkills = useCallback(async () => {
    invalidateSkillsCache();
    setSkillsLoading(true);
    try {
      const loaded = await getAllSkills();
      setSkills(loaded);
    } catch (err) {
      console.error("Failed to refresh skills:", err);
    } finally {
      setSkillsLoading(false);
    }
  }, []);

  const syncSkills = useCallback(async () => {
    setSkillsLoading(true);
    try {
      const result = await dataSyncSkills();
      const loaded = await getAllSkills();
      setSkills(loaded);
      return { success: result.success, skillCount: result.skillCount, error: result.error };
    } catch (err) {
      console.error("Failed to sync skills:", err);
      return { success: false, skillCount: 0, error: err instanceof Error ? err.message : "Unknown error" };
    } finally {
      setSkillsLoading(false);
    }
  }, []);

  const addSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => {
      if (prev.includes(skillId)) return prev;
      return [...prev, skillId];
    });
  }, []);

  const removeSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => prev.filter((id) => id !== skillId));
  }, []);

  const toggleSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => {
      if (prev.includes(skillId)) return prev.filter((id) => id !== skillId);
      return [...prev, skillId];
    });
  }, []);

  const clearSkills = useCallback(() => {
    setSelectedSkillIds([]);
  }, []);

  const addSuggestedSkills = useCallback((skillIds: string[]) => {
    setSelectedSkillIds((prev) => {
      const newIds = new Set(prev);
      for (const skillId of skillIds) {
        newIds.add(skillId);
      }
      return Array.from(newIds);
    });
  }, []);

  const getSelectedSkills = useCallback(() => {
    return skills.filter((s) => selectedSkillIds.includes(s.id));
  }, [selectedSkillIds, skills]);

  const buildSkillPayloads = useCallback(
    (skillIds: string[]): SkillPayload[] => {
      const payloads: SkillPayload[] = [];
      for (const id of skillIds) {
        const skill = skills.find((s) => s.id === id);
        if (!skill) continue;
        payloads.push({
          id: skill.id,
          name: skill.name,
          content: skill.content,
          key: `skill-${skill.id}`,
          label: skill.name,
          tags: skill.tags,
          targetToolId: skill.targetToolId,
        });
      }
      return payloads;
    },
    [skills]
  );

  // Build all slash commands
  const getAllCommands = useCallback(
    (templates: TemplateWithSource[]): SlashCommand[] => {
      const commands: SlashCommand[] = [
        { type: "template", id: "template", name: "/template", description: "Browse all templates", icon: "FileTemplate" },
        { type: "skill", id: "skill", name: "/skill", description: "Attach knowledge skills", icon: "BookOpen" },
        { type: "tool", id: "tool", name: "/tool", description: "Force a specific tool", icon: "Wrench" },
        { type: "search", id: "search", name: "/search", description: "Enable web search", icon: "Globe" },
        { type: "tool" as SlashCommandType, id: "suggestions", name: "/suggestions", description: "Toggle template suggestions panel", icon: "Lightbulb" },
        ...templates.map((t) => ({
          type: "direct-template" as const,
          id: t.id,
          name: `/${t.id}`,
          description: t.description,
          icon: t.icon,
          category: "Templates",
        })),
        ...skills.map((s) => ({
          type: "direct-skill" as const,
          id: s.id,
          name: `/${s.id}`,
          description: s.description,
          icon: s.icon,
          category: "Skills",
        })),
      ];
      return commands;
    },
    [skills]
  );

  const filterCommands = useCallback(
    (query: string, templates: TemplateWithSource[]): SlashCommand[] => {
      const allCommands = getAllCommands(templates);
      if (!query) return allCommands;

      const lowerQuery = query.toLowerCase();
      const scored = allCommands
        .map((cmd) => {
          const name = cmd.name.toLowerCase();
          const id = cmd.id.toLowerCase();
          const description = cmd.description.toLowerCase();

          let score = 0;
          if (name === `/${lowerQuery}` || id === lowerQuery) score = 100;
          else if (name.startsWith(`/${lowerQuery}`) || id.startsWith(lowerQuery)) score = 80;
          else if (name.includes(lowerQuery) || id.includes(lowerQuery)) score = 60;
          else if (description.startsWith(lowerQuery)) score = 40;
          else if (description.includes(lowerQuery)) score = 20;

          return { cmd, score };
        })
        .filter(({ score }) => score > 0)
        .sort((a, b) => b.score - a.score);

      return scored.map(({ cmd }) => cmd);
    },
    [getAllCommands]
  );

  return {
    skills,
    skillsLoading,
    refreshSkills,
    syncSkills,
    selectedSkillIds,
    addSkill,
    removeSkill,
    toggleSkill,
    clearSkills,
    addSuggestedSkills,
    getSelectedSkills,
    buildSkillPayloads,
    getAllCommands,
    filterCommands,
  };
}
