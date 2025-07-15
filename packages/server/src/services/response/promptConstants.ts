/**
 * Prompt Generation Constants
 * 
 * This file contains constants used by the PromptService for template
 * processing and prompt generation.
 */

/**
 * Default goal message when no specific goal is provided
 */
export const DEFAULT_GOAL = "Process current event.";

/**
 * Welcome message that prefixes all system prompts
 */
export const VROOLI_WELCOME_MESSAGE = "Welcome to Vrooli, a polymorphic, collaborative, and self-improving automation platform that helps you stay organized and achieve your goals.\\n\\n";

/**
 * Recruitment rule prompt for leader/coordinator/delegator roles
 * This is injected into {{ROLE_SPECIFIC_INSTRUCTIONS}} for qualifying roles
 */
export const RECRUITMENT_RULE_PROMPT = `## Recruitment rule:
If setting a new goal that spans multiple knowledge domains OR is estimated to exceed 2 hours OR 500 reasoning steps, you MUST add *all* of the following subtasks to the swarm's subtasks via the \`update_swarm_shared_state\` tool BEFORE any domain work:

[
  { "id":"T1", "description":"Look for a suitable existing team",
    "status":"todo" },
  { "id":"T2", "description":"If a team is found, set it as the swarm's team",
    "status":"todo", "depends_on":["T1"] },
  { "id":"T3", "description":"If not, create a new team for the task",
    "status":"todo", "depends_on":["T1"] },
  { "id":"T4", "description":"{{GOAL}}",
    "status":"todo", "depends_on":["T2","T3"] }
]`;

/**
 * Maximum length for swarm state sections before truncation
 */
export const MAX_STRING_PREVIEW_LENGTH = 2_000;

/**
 * Default fallback template when template file cannot be loaded
 */
export const DEFAULT_TEMPLATE_FALLBACK = "Your primary goal is: {{GOAL}}. Please act according to your role: {{ROLE}}. Critical: Prompt template file not found.";

/**
 * Standard template variables that can be used in prompt templates
 */
export const TEMPLATE_VARIABLES = {
    GOAL: "{{GOAL}}",
    ROLE: "{{ROLE}}",
    ROLE_UPPER: "{{ROLE | upper}}",
    BOT_ID: "{{BOT_ID}}",
    MEMBER_COUNT_LABEL: "{{MEMBER_COUNT_LABEL}}",
    ISO_EPOCH_SECONDS: "{{ISO_EPOCH_SECONDS}}",
    DISPLAY_DATE: "{{DISPLAY_DATE}}",
    ROLE_SPECIFIC_INSTRUCTIONS: "{{ROLE_SPECIFIC_INSTRUCTIONS}}",
    SWARM_STATE: "{{SWARM_STATE}}",
    TOOL_SCHEMAS: "{{TOOL_SCHEMAS}}",
} as const;

/**
 * Roles that should receive the recruitment rule instructions
 */
export const LEADERSHIP_ROLES = ["leader", "coordinator", "delegator"] as const;

/**
 * Default member count label for single-member scenarios
 */
export const DEFAULT_MEMBER_COUNT_LABEL = "1 member";

/**
 * Team-based member count label
 */
export const TEAM_BASED_MEMBER_COUNT_LABEL = "team-based swarm";
