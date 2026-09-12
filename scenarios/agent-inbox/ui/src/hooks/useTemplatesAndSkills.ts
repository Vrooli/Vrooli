/**
 * Hook for managing templates and skills state in the message composer.
 *
 * Composes useTemplateState and useSkillsState into a single interface.
 * See useTemplateState.ts and useSkillsState.ts for implementation details.
 *
 * Provides:
 * - Template selection and variable management
 * - Skill attachment/removal
 * - Slash command filtering
 * - Mode-based navigation for Suggestions
 * - Template CRUD operations (async, API-backed)
 * - State reset after message send
 */

import { useCallback } from "react";
import type {
  ActiveTemplate,
  Skill,
  SkillPayload,
  SlashCommand,
  Template,
  TemplateWithSource,
} from "@/lib/types/templates";
import type { CreateTemplateInput, UpdateTemplateInput, SkillResponse } from "@/lib/api";
import { useTemplateState } from "./useTemplateState";
import { useSkillsState } from "./useSkillsState";

export interface UseTemplatesAndSkillsReturn {
  // Data
  templates: TemplateWithSource[];
  skills: SkillResponse[];
  isLoading: boolean;
  skillsLoading: boolean;
  error: string | null;

  // Skills sync and refresh
  refreshSkills: () => Promise<void>;
  syncSkills: () => Promise<{ success: boolean; skillCount: number; error?: string }>;

  // Template state
  activeTemplate: ActiveTemplate | null;
  setActiveTemplate: (template: Template | null) => void;
  updateTemplateVariables: (values: Record<string, string>) => void;
  getFilledTemplateContent: () => string;
  clearTemplate: () => void;
  isTemplateValid: () => boolean;
  getTemplateMissingFields: () => string[];

  // Template CRUD (async)
  createTemplate: (
    template: CreateTemplateInput
  ) => Promise<TemplateWithSource | null>;
  updateTemplate: (
    id: string,
    updates: UpdateTemplateInput
  ) => Promise<TemplateWithSource | null>;
  deleteTemplate: (id: string) => Promise<boolean>;
  resetTemplate: (id: string) => Promise<TemplateWithSource | null>;
  refreshTemplates: () => Promise<void>;

  // Mode navigation
  currentModePath: string[];
  setCurrentModePath: (path: string[]) => void;
  getTemplatesAtPath: (path: string[]) => TemplateWithSource[];
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
  buildSkillPayloads: (skillIds: string[]) => SkillPayload[];

  // Slash commands
  getAllCommands: () => SlashCommand[];
  filterCommands: (query: string) => SlashCommand[];

  // Reset all state (called after message send)
  resetAll: () => void;
}

export function useTemplatesAndSkills(): UseTemplatesAndSkillsReturn {
  const templateState = useTemplateState();
  const skillsState = useSkillsState();
  const {
    activeTemplate,
    clearTemplate,
    createTemplate,
    currentModePath,
    deleteTemplate,
    error,
    getFilledTemplateContent,
    getSubmodesAtPath,
    getTemplateMissingFields,
    getTemplatesAtPath,
    isLoading,
    isTemplateValid,
    navigateBack,
    navigateToMode,
    refreshTemplates,
    resetModePath,
    resetTemplate,
    setActiveTemplate: setTemplateActive,
    setCurrentModePath,
    templates,
    updateTemplate,
    updateTemplateVariables,
  } = templateState;
  const {
    addSkill,
    addSuggestedSkills,
    buildSkillPayloads,
    clearSkills,
    filterCommands: filterSkillCommands,
    getAllCommands: getAllSkillCommands,
    getSelectedSkills,
    refreshSkills,
    removeSkill,
    selectedSkillIds,
    skills,
    skillsLoading,
    syncSkills,
    toggleSkill,
  } = skillsState;

  // Wrap setActiveTemplate to auto-attach suggested skills
  const setActiveTemplate = useCallback(
    (template: Template | null) => {
      setTemplateActive(template, (skillIds) => {
        addSuggestedSkills(skillIds);
      });
    },
    [addSuggestedSkills, setTemplateActive]
  );

  // Wrap slash command methods to pass templates automatically
  const getAllCommands = useCallback(
    () => getAllSkillCommands(templates),
    [getAllSkillCommands, templates]
  );

  const filterCommands = useCallback(
    (query: string) => filterSkillCommands(query, templates),
    [filterSkillCommands, templates]
  );

  // Reset all state
  const resetAll = useCallback(() => {
    clearTemplate();
    clearSkills();
  }, [clearSkills, clearTemplate]);

  return {
    // Data
    templates,
    skills,
    isLoading,
    skillsLoading,
    error,

    // Skills sync and refresh
    refreshSkills,
    syncSkills,

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
    resetTemplate,
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
    buildSkillPayloads,

    // Slash commands
    getAllCommands,
    filterCommands,

    // Reset
    resetAll,
  };
}
