/**
 * Types for Templates and Skills feature.
 *
 * Templates: Message scaffolding with variables - help construct the "what to do" instruction.
 * Skills: Knowledge modules injected into context - provide the "how to do it" methodology.
 */

/** Variable definition for template placeholders */
export interface TemplateVariable {
  name: string;
  label: string;
  type: "text" | "textarea" | "select";
  placeholder?: string;
  options?: string[]; // For select type
  required?: boolean;
  defaultValue?: string;
}

/** Template - Message scaffolding with variables */
export interface Template {
  id: string;
  name: string;
  description: string;
  icon?: string; // Lucide icon name
  category?: string;
  content: string; // Template text with {{variable}} placeholders
  variables: TemplateVariable[];
  suggestedSkillIds?: string[]; // Skills that auto-attach when template selected
  suggestedToolIds?: string[]; // Tools suggested for this template (e.g., ["spawn_coding_agent"])
  modes?: string[]; // Hierarchical path: ["Research", "Codebase Structure"]
  isBuiltIn?: boolean; // Distinguishes system vs user templates
  createdAt?: string; // ISO timestamp (user templates)
  updatedAt?: string; // ISO timestamp (user templates)
}

/** Skill - Knowledge module injected into context */
export interface Skill {
  id: string;
  name: string;
  description: string;
  icon?: string; // Lucide icon name
  category?: string;
  content: string; // Full methodology/knowledge content
  tags?: string[];
}

/**
 * SkillPayload - Full skill data sent to backend for tool context injection.
 *
 * This is converted to ContextAttachment format in agent-manager:
 * - type: "skill"
 * - key: skill ID
 * - label: skill name
 * - content: full skill content
 * - tags: skill tags
 */
export interface SkillPayload {
  id: string;
  name: string;
  content: string;
  key: string; // Unique identifier (typically "skill-{id}")
  label: string; // Display label (typically same as name)
  tags?: string[];
  targetToolId?: string; // If set, skill only sent to this specific tool
}

/** Active template state with filled variables */
export interface ActiveTemplate {
  template: Template;
  variableValues: Record<string, string>;
}

/** Slash command types */
export type SlashCommandType =
  | "template"
  | "skill"
  | "tool"
  | "search"
  | "direct-template"
  | "direct-skill";

export interface SlashCommand {
  type: SlashCommandType;
  id: string;
  name: string;
  description: string;
  icon?: string;
  category?: string;
}

/** Mode history entry for frecency-based ordering */
export interface ModeHistoryEntry {
  path: string; // Serialized path like "Research" or "Research/Codebase Structure"
  count: number; // Usage count
  lastUsed: string; // ISO timestamp
}

/** Suggestions settings stored in localStorage */
export interface SuggestionsSettings {
  visible: boolean;
  mergeModel: string; // Model ID for AI merge feature
}

/** AI merge action options */
export type MergeAction = "overwrite" | "merge" | "cancel";

/** Top-level suggestion modes */
export const SUGGESTION_MODES = [
  "Research",
  "Debug/Fix",
  "Implement Feature",
  "Refactor",
  "Write Tests",
  "Explain/Teach",
  "Review/QA",
] as const;

export type SuggestionMode = (typeof SUGGESTION_MODES)[number];
