import { useCallback, useMemo } from "react";
import type { ComponentType, SVGProps } from "react";
import * as LucideIcons from "lucide-react";
import { BookOpen } from "lucide-react";

// Type for Lucide icon components
export type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

// Get icon component from name
export function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

// Form state for tracking changes
export interface SkillFormState {
  name: string;
  description: string;
  icon: string;
  modes: string[];
  content: string;
  tagsInput: string;
  targetToolId: string;
  draft: boolean;
}

// Validate form fields, returns error map (empty = valid)
export function validateSkillForm(state: SkillFormState): Record<string, string> {
  const errors: Record<string, string> = {};

  if (!state.name.trim()) {
    errors.name = "Name is required";
  }
  if (!state.description.trim()) {
    errors.description = "Description is required";
  }
  if (!state.content.trim()) {
    errors.content = "Content is required";
  }

  return errors;
}

// Parse tags from comma-separated string
export function parseTags(input: string): string[] {
  return input
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

// Build skill data payload from form state
export function buildSkillData(state: SkillFormState) {
  const tags = parseTags(state.tagsInput);

  return {
    name: state.name.trim(),
    description: state.description.trim(),
    icon: state.icon,
    modes: state.modes.length > 0 ? state.modes : undefined,
    content: state.content.trim(),
    tags: tags.length > 0 ? tags : undefined,
    targetToolId: state.targetToolId.trim() || undefined,
    draft: state.draft || undefined,
  };
}

// Hook: check if the form has unsaved changes compared to the original skill
export function useHasUnsavedChanges(
  readOnly: boolean,
  skill: { name: string; description: string; icon?: string; modes?: string[]; content: string; tags?: string[]; targetToolId?: string; draft?: boolean } | undefined,
  state: SkillFormState
): boolean {
  return useMemo(() => {
    if (readOnly) return false;
    if (!skill) {
      return !!(
        state.name.trim() ||
        state.description.trim() ||
        state.content.trim() ||
        state.tagsInput.trim() ||
        state.targetToolId.trim() ||
        state.modes.length > 0 ||
        state.draft
      );
    }
    const originalTags = skill.tags?.join(", ") || "";
    return (
      state.name !== skill.name ||
      state.description !== skill.description ||
      state.icon !== (skill.icon || "BookOpen") ||
      JSON.stringify(state.modes) !== JSON.stringify(skill.modes || []) ||
      state.content !== skill.content ||
      state.tagsInput !== originalTags ||
      state.targetToolId !== (skill.targetToolId || "") ||
      state.draft !== (skill.draft || false)
    );
  }, [readOnly, skill, state]);
}

// Hook: get current form state as an object
export function useGetCurrentFormState(state: SkillFormState): () => SkillFormState {
  return useCallback(
    () => ({ ...state }),
    [state]
  );
}
