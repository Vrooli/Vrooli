/**
 * Hook for managing templates and skills state in the message composer.
 *
 * Provides:
 * - Template selection and variable management
 * - Skill attachment/removal
 * - Slash command filtering
 * - State reset after message send
 */

import { useCallback, useMemo, useState } from "react";
import {
  TEMPLATES,
  fillTemplateContent,
  validateTemplateVariables,
} from "@/data/templates";
import { SKILLS, getSkillsByIds } from "@/data/skills";
import type {
  ActiveTemplate,
  Skill,
  SlashCommand,
  Template,
} from "@/lib/types/templates";

export interface UseTemplatesAndSkillsReturn {
  // Data
  templates: Template[];
  skills: Skill[];

  // Template state
  activeTemplate: ActiveTemplate | null;
  setActiveTemplate: (template: Template | null) => void;
  updateTemplateVariables: (values: Record<string, string>) => void;
  getFilledTemplateContent: () => string;
  clearTemplate: () => void;
  isTemplateValid: () => boolean;
  getTemplateMissingFields: () => string[];

  // Skills state
  selectedSkillIds: string[];
  addSkill: (skillId: string) => void;
  removeSkill: (skillId: string) => void;
  toggleSkill: (skillId: string) => void;
  clearSkills: () => void;
  getSelectedSkills: () => Skill[];

  // Slash commands
  getAllCommands: () => SlashCommand[];
  filterCommands: (query: string) => SlashCommand[];

  // Reset all state (called after message send)
  resetAll: () => void;
}

export function useTemplatesAndSkills(): UseTemplatesAndSkillsReturn {
  // Template state
  const [activeTemplate, setActiveTemplateState] =
    useState<ActiveTemplate | null>(null);

  // Skills state
  const [selectedSkillIds, setSelectedSkillIds] = useState<string[]>([]);

  // Set active template with default variable values
  const setActiveTemplate = useCallback(
    (template: Template | null) => {
      if (!template) {
        setActiveTemplateState(null);
        return;
      }

      // Initialize variable values with defaults
      const variableValues: Record<string, string> = {};
      for (const variable of template.variables) {
        variableValues[variable.name] = variable.defaultValue || "";
      }

      setActiveTemplateState({ template, variableValues });

      // Auto-attach suggested skills
      if (template.suggestedSkillIds?.length) {
        setSelectedSkillIds((prev) => {
          const newIds = new Set(prev);
          for (const skillId of template.suggestedSkillIds!) {
            newIds.add(skillId);
          }
          return Array.from(newIds);
        });
      }
    },
    []
  );

  // Update template variable values
  const updateTemplateVariables = useCallback(
    (values: Record<string, string>) => {
      setActiveTemplateState((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          variableValues: { ...prev.variableValues, ...values },
        };
      });
    },
    []
  );

  // Get filled template content
  const getFilledTemplateContent = useCallback(() => {
    if (!activeTemplate) return "";
    return fillTemplateContent(
      activeTemplate.template,
      activeTemplate.variableValues
    );
  }, [activeTemplate]);

  // Clear active template
  const clearTemplate = useCallback(() => {
    setActiveTemplateState(null);
  }, []);

  // Check if template is valid (all required fields filled)
  const isTemplateValid = useCallback(() => {
    if (!activeTemplate) return true; // No template = valid
    const { valid } = validateTemplateVariables(
      activeTemplate.template,
      activeTemplate.variableValues
    );
    return valid;
  }, [activeTemplate]);

  // Get missing required fields
  const getTemplateMissingFields = useCallback(() => {
    if (!activeTemplate) return [];
    const { missingFields } = validateTemplateVariables(
      activeTemplate.template,
      activeTemplate.variableValues
    );
    return missingFields;
  }, [activeTemplate]);

  // Add skill
  const addSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => {
      if (prev.includes(skillId)) return prev;
      return [...prev, skillId];
    });
  }, []);

  // Remove skill
  const removeSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => prev.filter((id) => id !== skillId));
  }, []);

  // Toggle skill
  const toggleSkill = useCallback((skillId: string) => {
    setSelectedSkillIds((prev) => {
      if (prev.includes(skillId)) {
        return prev.filter((id) => id !== skillId);
      }
      return [...prev, skillId];
    });
  }, []);

  // Clear all skills
  const clearSkills = useCallback(() => {
    setSelectedSkillIds([]);
  }, []);

  // Get selected skill objects
  const getSelectedSkills = useCallback(() => {
    return getSkillsByIds(selectedSkillIds);
  }, [selectedSkillIds]);

  // Build all slash commands
  const getAllCommands = useCallback((): SlashCommand[] => {
    const commands: SlashCommand[] = [
      // Category commands
      {
        type: "template",
        id: "template",
        name: "/template",
        description: "Browse all templates",
        icon: "FileTemplate",
      },
      {
        type: "skill",
        id: "skill",
        name: "/skill",
        description: "Attach knowledge skills",
        icon: "BookOpen",
      },
      {
        type: "tool",
        id: "tool",
        name: "/tool",
        description: "Force a specific tool",
        icon: "Wrench",
      },
      {
        type: "search",
        id: "search",
        name: "/search",
        description: "Enable web search",
        icon: "Globe",
      },

      // Direct template commands
      ...TEMPLATES.map((t) => ({
        type: "direct-template" as const,
        id: t.id,
        name: `/${t.id}`,
        description: t.description,
        icon: t.icon,
        category: "Templates",
      })),

      // Direct skill commands
      ...SKILLS.map((s) => ({
        type: "direct-skill" as const,
        id: s.id,
        name: `/${s.id}`,
        description: s.description,
        icon: s.icon,
        category: "Skills",
      })),
    ];

    return commands;
  }, []);

  // Filter commands by query
  const filterCommands = useCallback(
    (query: string): SlashCommand[] => {
      const allCommands = getAllCommands();
      if (!query) return allCommands;

      const lowerQuery = query.toLowerCase();
      return allCommands.filter(
        (cmd) =>
          cmd.name.toLowerCase().includes(lowerQuery) ||
          cmd.description.toLowerCase().includes(lowerQuery) ||
          cmd.id.toLowerCase().includes(lowerQuery)
      );
    },
    [getAllCommands]
  );

  // Reset all state
  const resetAll = useCallback(() => {
    setActiveTemplateState(null);
    setSelectedSkillIds([]);
  }, []);

  return {
    // Data
    templates: TEMPLATES,
    skills: SKILLS,

    // Template state
    activeTemplate,
    setActiveTemplate,
    updateTemplateVariables,
    getFilledTemplateContent,
    clearTemplate,
    isTemplateValid,
    getTemplateMissingFields,

    // Skills state
    selectedSkillIds,
    addSkill,
    removeSkill,
    toggleSkill,
    clearSkills,
    getSelectedSkills,

    // Slash commands
    getAllCommands,
    filterCommands,

    // Reset
    resetAll,
  };
}
