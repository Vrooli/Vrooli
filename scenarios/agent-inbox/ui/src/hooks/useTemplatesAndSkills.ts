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

  // Wrap setActiveTemplate to auto-attach suggested skills
  const setActiveTemplate = useCallback(
    (template: Template | null) => {
      templateState.setActiveTemplate(template, (skillIds) => {
        skillsState.addSuggestedSkills(skillIds);
      });
    },
    [templateState.setActiveTemplate, skillsState.addSuggestedSkills]
  );

  // Wrap slash command methods to pass templates automatically
  const getAllCommands = useCallback(
    () => skillsState.getAllCommands(templateState.templates),
    [skillsState.getAllCommands, templateState.templates]
  );

  const filterCommands = useCallback(
    (query: string) => skillsState.filterCommands(query, templateState.templates),
    [skillsState.filterCommands, templateState.templates]
  );

  // Reset all state
  const resetAll = useCallback(() => {
    templateState.clearTemplate();
    skillsState.clearSkills();
  }, [templateState.clearTemplate, skillsState.clearSkills]);

  return {
    // Data
    templates: templateState.templates,
    skills: skillsState.skills,
    isLoading: templateState.isLoading,
    skillsLoading: skillsState.skillsLoading,
    error: templateState.error,

    // Skills sync and refresh
    refreshSkills: skillsState.refreshSkills,
    syncSkills: skillsState.syncSkills,

    // Template state
    activeTemplate: templateState.activeTemplate,
    setActiveTemplate,
    updateTemplateVariables: templateState.updateTemplateVariables,
    getFilledTemplateContent: templateState.getFilledTemplateContent,
    clearTemplate: templateState.clearTemplate,
    isTemplateValid: templateState.isTemplateValid,
    getTemplateMissingFields: templateState.getTemplateMissingFields,

    // Template CRUD
    createTemplate: templateState.createTemplate,
    updateTemplate: templateState.updateTemplate,
    deleteTemplate: templateState.deleteTemplate,
    resetTemplate: templateState.resetTemplate,
    refreshTemplates: templateState.refreshTemplates,

    // Mode navigation
    currentModePath: templateState.currentModePath,
    setCurrentModePath: templateState.setCurrentModePath,
    getTemplatesAtPath: templateState.getTemplatesAtPath,
    getSubmodesAtPath: templateState.getSubmodesAtPath,
    navigateToMode: templateState.navigateToMode,
    navigateBack: templateState.navigateBack,
    resetModePath: templateState.resetModePath,

    // Skills state
    selectedSkillIds: skillsState.selectedSkillIds,
    addSkill: skillsState.addSkill,
    removeSkill: skillsState.removeSkill,
    toggleSkill: skillsState.toggleSkill,
    clearSkills: skillsState.clearSkills,
    getSelectedSkills: skillsState.getSelectedSkills,
    buildSkillPayloads: skillsState.buildSkillPayloads,

    // Slash commands
    getAllCommands,
    filterCommands,

    // Reset
    resetAll,
  };
}
