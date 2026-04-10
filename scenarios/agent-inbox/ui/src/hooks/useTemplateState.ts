/**
 * useTemplateState - Template loading, CRUD, active template state, and mode navigation.
 *
 * Extracted from useTemplatesAndSkills.ts for modularity.
 */

import { useCallback, useEffect, useState } from "react";
import {
  getAllTemplates,
  fillTemplateContent,
  validateTemplateVariables,
  createTemplate as createTemplateAPI,
  updateTemplate as updateTemplateAPI,
  deleteTemplate as deleteTemplateAPI,
  resetTemplate as resetTemplateAPI,
  invalidateTemplatesCache,
  migrateLegacyTemplates,
  hasLegacyData,
} from "@/data/templates";
import type {
  ActiveTemplate,
  Template,
  TemplateWithSource,
} from "@/lib/types/templates";
import type { CreateTemplateInput, UpdateTemplateInput } from "@/lib/api";

export interface UseTemplateStateReturn {
  // Data
  templates: TemplateWithSource[];
  isLoading: boolean;
  error: string | null;

  // Template state
  activeTemplate: ActiveTemplate | null;
  setActiveTemplate: (template: Template | null, onSkillsSuggested?: (skillIds: string[]) => void) => void;
  updateTemplateVariables: (values: Record<string, string>) => void;
  getFilledTemplateContent: () => string;
  clearTemplate: () => void;
  isTemplateValid: () => boolean;
  getTemplateMissingFields: () => string[];

  // Template CRUD (async)
  createTemplate: (template: CreateTemplateInput) => Promise<TemplateWithSource | null>;
  updateTemplate: (id: string, updates: UpdateTemplateInput) => Promise<TemplateWithSource | null>;
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
}

export function useTemplateState(): UseTemplateStateReturn {
  const [templates, setTemplates] = useState<TemplateWithSource[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTemplate, setActiveTemplateState] = useState<ActiveTemplate | null>(null);
  const [currentModePath, setCurrentModePath] = useState<string[]>([]);

  // Load templates on mount
  useEffect(() => {
    let mounted = true;

    async function loadTemplates() {
      setIsLoading(true);
      setError(null);

      try {
        if (hasLegacyData()) {
          const migrated = await migrateLegacyTemplates();
          if (migrated > 0) {
            console.log(`Migrated ${migrated} legacy templates to file storage`);
          }
        }

        const loaded = await getAllTemplates();
        if (mounted) setTemplates(loaded);
      } catch (err) {
        if (mounted) setError(err instanceof Error ? err.message : "Failed to load templates");
      } finally {
        if (mounted) setIsLoading(false);
      }
    }

    loadTemplates();
    return () => { mounted = false; };
  }, []);

  const refreshTemplates = useCallback(async () => {
    invalidateTemplatesCache();
    setIsLoading(true);
    setError(null);
    try {
      const loaded = await getAllTemplates();
      setTemplates(loaded);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load templates");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Template CRUD
  const createTemplate = useCallback(
    async (template: CreateTemplateInput): Promise<TemplateWithSource | null> => {
      try {
        const newTemplate = await createTemplateAPI(template);
        await refreshTemplates();
        return newTemplate;
      } catch (err) {
        console.error("Failed to create template:", err);
        return null;
      }
    },
    [refreshTemplates]
  );

  const updateTemplate = useCallback(
    async (id: string, updates: UpdateTemplateInput): Promise<TemplateWithSource | null> => {
      try {
        const updated = await updateTemplateAPI(id, updates);
        await refreshTemplates();
        return updated;
      } catch (err) {
        console.error("Failed to update template:", err);
        return null;
      }
    },
    [refreshTemplates]
  );

  const deleteTemplate = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const deleted = await deleteTemplateAPI(id);
        if (deleted) await refreshTemplates();
        return deleted;
      } catch (err) {
        console.error("Failed to delete template:", err);
        return false;
      }
    },
    [refreshTemplates]
  );

  const resetTemplate = useCallback(
    async (id: string): Promise<TemplateWithSource | null> => {
      try {
        const reset = await resetTemplateAPI(id);
        if (reset) await refreshTemplates();
        return reset;
      } catch (err) {
        console.error("Failed to reset template:", err);
        return null;
      }
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

  const getTemplatesAtPath = useCallback(
    (path: string[]): TemplateWithSource[] => {
      if (path.length === 0) return [];
      return templates.filter((t) => {
        if (!t.modes || t.modes.length === 0) return false;
        if (t.modes.length < path.length) return false;
        for (let i = 0; i < path.length; i++) {
          if (t.modes[i] !== path[i]) return false;
        }
        return t.modes.length === path.length;
      });
    },
    [templates]
  );

  const getSubmodesAtPath = useCallback(
    (path: string[]): string[] => {
      const submodes = new Set<string>();
      templates.forEach((t) => {
        if (!t.modes || t.modes.length <= path.length) return;
        for (let i = 0; i < path.length; i++) {
          if (t.modes[i] !== path[i]) return;
        }
        const nextMode = t.modes[path.length];
        if (nextMode) submodes.add(nextMode);
      });
      return Array.from(submodes).sort();
    },
    [templates]
  );

  // Set active template with default variable values
  const setActiveTemplate = useCallback(
    (template: Template | null, onSkillsSuggested?: (skillIds: string[]) => void) => {
      if (!template) {
        setActiveTemplateState(null);
        return;
      }

      const variableValues: Record<string, string> = {};
      for (const variable of template.variables) {
        variableValues[variable.name] = variable.defaultValue || "";
      }

      setActiveTemplateState({ template, variableValues });

      // Notify about suggested skills
      const suggestedSkills = template.suggestedSkillIds;
      if (suggestedSkills?.length && onSkillsSuggested) {
        onSkillsSuggested(suggestedSkills);
      }
    },
    []
  );

  const updateTemplateVariables = useCallback(
    (values: Record<string, string>) => {
      setActiveTemplateState((prev) => {
        if (!prev) return null;
        return { ...prev, variableValues: { ...prev.variableValues, ...values } };
      });
    },
    []
  );

  const getFilledTemplateContent = useCallback(() => {
    if (!activeTemplate) return "";
    return fillTemplateContent(activeTemplate.template, activeTemplate.variableValues);
  }, [activeTemplate]);

  const clearTemplate = useCallback(() => {
    setActiveTemplateState(null);
  }, []);

  const isTemplateValid = useCallback(() => {
    if (!activeTemplate) return true;
    const { valid } = validateTemplateVariables(activeTemplate.template, activeTemplate.variableValues);
    return valid;
  }, [activeTemplate]);

  const getTemplateMissingFields = useCallback(() => {
    if (!activeTemplate) return [];
    const { missingFields } = validateTemplateVariables(activeTemplate.template, activeTemplate.variableValues);
    return missingFields;
  }, [activeTemplate]);

  return {
    templates,
    isLoading,
    error,
    activeTemplate,
    setActiveTemplate,
    updateTemplateVariables,
    getFilledTemplateContent,
    clearTemplate,
    isTemplateValid,
    getTemplateMissingFields,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    resetTemplate,
    refreshTemplates,
    currentModePath,
    setCurrentModePath,
    getTemplatesAtPath,
    getSubmodesAtPath,
    navigateToMode,
    navigateBack,
    resetModePath,
  };
}
