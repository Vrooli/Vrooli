/**
 * Type definitions and constants for the TemplateEditorModal.
 */

import type { ComponentType, SVGProps } from "react";
import type { Template, TemplateVariable, TemplateSource, TemplateWithSource } from "@/lib/types/templates";

// Type for Lucide icon components
export type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

export interface SaveOptions {
  applyToDefault?: boolean;
}

export interface TemplateEditorModalProps {
  open: boolean;
  onClose: () => void;
  template?: Template; // Undefined for create, defined for edit
  templateSource?: TemplateSource; // Source of the template being edited
  defaultModes?: string[]; // Pre-fill modes when creating from Suggestions
  onSave?: (
    template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
    options?: SaveOptions
  ) => void;
  readOnly?: boolean; // If true, modal is in preview mode (no editing)
  onEdit?: () => void; // Callback when Edit button is clicked in readOnly mode
  // Multi-item mode props
  allTemplates?: TemplateWithSource[]; // If provided, shows tree sidebar for navigation
  onSelectTemplate?: (template: TemplateWithSource) => void; // Called when switching templates
  onSaveAll?: (templates: Array<{ id: string; data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">; options?: SaveOptions }>) => Promise<void>;
}

// Form state for tracking changes
export interface TemplateFormState {
  name: string;
  description: string;
  icon: string;
  modes: string[];
  content: string;
  variables: TemplateVariable[];
  selectedToolIds: string[];
  applyToDefault: boolean;
  draft: boolean;
}

export const VARIABLE_TYPES: TemplateVariable["type"][] = [
  "text",
  "textarea",
  "select",
];
