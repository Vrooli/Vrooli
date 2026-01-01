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
