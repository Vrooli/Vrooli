/**
 * Hook for managing templates and skills state in the message composer.
 *
 * Provides:
 * - Template selection and variable management
 * - Skill attachment/removal
 * - Slash command filtering
 * - Mode-based navigation for Suggestions
 * - Template CRUD operations
 * - State reset after message send
 */

import { useCallback, useMemo, useState } from "react";
import {
  getAllTemplates,
  fillTemplateContent,
  validateTemplateVariables,
  createTemplate as createTemplateStorage,
  updateTemplate as updateTemplateStorage,
  deleteTemplate as deleteTemplateStorage,
  hideBuiltInTemplate,
  unhideBuiltInTemplate,
} from "@/data/templates";
import { SKILLS, getSkillsByIds } from "@/data/skills";
import type {
  ActiveTemplate,
  Skill,
  SlashCommand,
  SlashCommandType,
  Template,
} from "@/lib/types/templates";
import { serializeModePath } from "@/lib/frecency";

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

  // Template CRUD
  createTemplate: (
    template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">
  ) => Template;
  updateTemplate: (id: string, updates: Partial<Template>) => Template | null;
  deleteTemplate: (id: string) => boolean;
  hideTemplate: (id: string) => void;
  unhideTemplate: (id: string) => void;
  refreshTemplates: () => void;

  // Mode navigation
  currentModePath: string[];
  setCurrentModePath: (path: string[]) => void;
  getTemplatesAtPath: (path: string[]) => Template[];
  getSubmodesAtPath: (path: string[]) => string[];
  navigateToMode: (mode: string) => void;
  navigateBack: () => void;
  resetModePath: () => void;

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
  // Template list state (refreshable)
  const [templates, setTemplates] = useState<Template[]>(getAllTemplates);

  // Active template state
  const [activeTemplate, setActiveTemplateState] =
    useState<ActiveTemplate | null>(null);

  // Mode navigation state
  const [currentModePath, setCurrentModePath] = useState<string[]>([]);

  // Skills state
  const [selectedSkillIds, setSelectedSkillIds] = useState<string[]>([]);

  // Refresh templates from storage
  const refreshTemplates = useCallback(() => {
    setTemplates(getAllTemplates());
  }, []);

  // Template CRUD operations
  const createTemplate = useCallback(
    (
      template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">
    ): Template => {
      const newTemplate = createTemplateStorage(template);
      refreshTemplates();
      return newTemplate;
    },
    [refreshTemplates]
  );

  const updateTemplate = useCallback(
    (id: string, updates: Partial<Template>): Template | null => {
      const updated = updateTemplateStorage(id, updates);
      if (updated) {
        refreshTemplates();
      }
      return updated;
    },
    [refreshTemplates]
  );

  const deleteTemplate = useCallback(
    (id: string): boolean => {
      const deleted = deleteTemplateStorage(id);
      if (deleted) {
        refreshTemplates();
      }
      return deleted;
    },
    [refreshTemplates]
  );

  const hideTemplate = useCallback(
    (id: string): void => {
      hideBuiltInTemplate(id);
      refreshTemplates();
    },
    [refreshTemplates]
  );

  const unhideTemplate = useCallback(
    (id: string): void => {
      unhideBuiltInTemplate(id);
      refreshTemplates();
    },
    [refreshTemplates]
  );

  // Mode navigation
  const navigateToMode = useCallback((mode: string) => {
    setCurrentModePath((prev) => [...prev, mode]);
  }, []);

  const navigateBack = useCallback(() => {
    setCurrentModePath((prev) => prev.slice(0, -1));
  }, []);

  const resetModePath = useCallback(() => {
    setCurrentModePath([]);
  }, []);

  // Get templates at a specific path
  const getTemplatesAtPath = useCallback(
    (path: string[]): Template[] => {
      if (path.length === 0) {
        // At root, no templates - only mode chips
        return [];
      }

      return templates.filter((t) => {
        if (!t.modes || t.modes.length === 0) return false;

        // Check if template's modes match the path exactly or extend it
        if (t.modes.length < path.length) return false;

        // All path elements must match the template's modes
        for (let i = 0; i < path.length; i++) {
          if (t.modes[i] !== path[i]) return false;
        }

        // Template must be exactly at this level (modes.length === path.length)
        // or this is the deepest level of the template
        return t.modes.length === path.length;
      });
    },
    [templates]
  );

  // Get submodes at a specific path
  const getSubmodesAtPath = useCallback(
    (path: string[]): string[] => {
      const submodes = new Set<string>();

      templates.forEach((t) => {
        if (!t.modes || t.modes.length <= path.length) return;

        // Check if template's modes match the path
        for (let i = 0; i < path.length; i++) {
          if (t.modes[i] !== path[i]) return;
        }

        // Add the next level mode
        submodes.add(t.modes[path.length]);
      });

      return Array.from(submodes).sort();
    },
    [templates]
  );

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
      // Suggestions toggle command
      {
        type: "tool" as SlashCommandType,
        id: "suggestions",
        name: "/suggestions",
        description: "Toggle template suggestions panel",
        icon: "Lightbulb",
      },

      // Direct template commands
      ...templates.map((t) => ({
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
  }, [templates]);

  // Filter and sort commands by query relevance
  const filterCommands = useCallback(
    (query: string): SlashCommand[] => {
      const allCommands = getAllCommands();
      if (!query) return allCommands;

      const lowerQuery = query.toLowerCase();

      // Score each command by match quality
      const scored = allCommands
        .map((cmd) => {
          const name = cmd.name.toLowerCase();
          const id = cmd.id.toLowerCase();
          const description = cmd.description.toLowerCase();

          let score = 0;

          // Exact match on name or id (highest priority)
          if (name === `/${lowerQuery}` || id === lowerQuery) {
            score = 100;
          }
          // Name or id starts with query
          else if (name.startsWith(`/${lowerQuery}`) || id.startsWith(lowerQuery)) {
            score = 80;
          }
          // Name or id contains query
          else if (name.includes(lowerQuery) || id.includes(lowerQuery)) {
            score = 60;
          }
          // Description starts with query
          else if (description.startsWith(lowerQuery)) {
            score = 40;
          }
          // Description contains query
          else if (description.includes(lowerQuery)) {
            score = 20;
          }

          return { cmd, score };
        })
        .filter(({ score }) => score > 0)
        .sort((a, b) => b.score - a.score);

      return scored.map(({ cmd }) => cmd);
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
    templates,
    skills: SKILLS,

    // Template state
    activeTemplate,
    setActiveTemplate,
    updateTemplateVariables,
    getFilledTemplateContent,
    clearTemplate,
    isTemplateValid,
    getTemplateMissingFields,

    // Template CRUD
    createTemplate,
    updateTemplate,
    deleteTemplate,
    hideTemplate,
    unhideTemplate,
    refreshTemplates,

    // Mode navigation
    currentModePath,
    setCurrentModePath,
    getTemplatesAtPath,
    getSubmodesAtPath,
    navigateToMode,
    navigateBack,
    resetModePath,

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
