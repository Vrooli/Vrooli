import { type Logger } from "winston";
import * as fs from "fs/promises";
import * as path from "path";
import { fileURLToPath } from "url";
import { SECONDS_1_MS } from "@vrooli/shared";
import { type SwarmState, type SwarmAgent, type SwarmConfig } from "@vrooli/shared";
import { ToolRegistry } from "../../../mcp/registry.js";

/**
 * Prompt Template Service for Tier 1
 * 
 * Loads and processes the prompt.txt template to give agents their context
 * and instructions. This enables emergent behavior through natural language
 * reasoning rather than hard-coded logic.
 */
export class PromptTemplateService {
    private readonly logger: Logger;
    private readonly toolRegistry: ToolRegistry;
    private cachedTemplate?: string;
    private cacheTime?: number;
    private readonly CACHE_DURATION_MS = 60000; // 1 minute cache

    constructor(logger: Logger) {
        this.logger = logger;
        this.toolRegistry = new ToolRegistry(logger);
    }

    /**
     * Loads the prompt template from disk with caching
     */
    async loadTemplate(): Promise<string> {
        // Check cache
        if (this.cachedTemplate && this.cacheTime && 
            Date.now() - this.cacheTime < this.CACHE_DURATION_MS) {
            return this.cachedTemplate;
        }

        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        // Navigate to conversation directory where prompt.txt lives
        const promptPath = path.join(__dirname, "../../../conversation/prompt.txt");

        try {
            this.logger.debug("[PromptTemplateService] Loading prompt template", { path: promptPath });
            this.cachedTemplate = await fs.readFile(promptPath, "utf-8");
            this.cacheTime = Date.now();
            return this.cachedTemplate;
        } catch (error) {
            this.logger.error("[PromptTemplateService] Failed to load prompt template", {
                error: error instanceof Error ? error.message : String(error),
                path: promptPath,
            });
            // Return a minimal fallback template
            return this.getFallbackTemplate();
        }
    }

    /**
     * Process template with agent-specific context
     */
    processTemplate(
        template: string,
        context: {
            ROLE: string;
            GOAL: string;
            MEMBER_COUNT_LABEL?: string;
            SWARM_STATE: string;
            TOOL_SCHEMAS: string;
            BOT_ID: string;
            ROLE_SPECIFIC_INSTRUCTIONS?: string;
        },
    ): string {
        let processed = template;

        // Replace all template variables
        processed = processed.replace(/\{\{ROLE\s*\|\s*upper\}\}/g, context.ROLE.toUpperCase());
        processed = processed.replace(/\{\{ROLE\}\}/g, context.ROLE);
        processed = processed.replace(/\{\{GOAL(?:\s*:-[^}]+)?\}\}/g, context.GOAL);
        processed = processed.replace(/\{\{MEMBER_COUNT_LABEL(?:\s*:-[^}]+)?\}\}/g, 
            context.MEMBER_COUNT_LABEL || "1 member");
        processed = processed.replace(/\{\{ISO_EPOCH_SECONDS\}\}/g, 
            Math.floor(Date.now() / SECONDS_1_MS).toString());
        processed = processed.replace(/\{\{DISPLAY_DATE\}\}/g, 
            new Date().toLocaleString());
        processed = processed.replace(/\{\{BOT_ID\}\}/g, context.BOT_ID);
        processed = processed.replace(/\{\{SWARM_STATE\}\}/g, context.SWARM_STATE);
        processed = processed.replace(/\{\{TOOL_SCHEMAS\}\}/g, context.TOOL_SCHEMAS);
        processed = processed.replace(/\{\{ROLE_SPECIFIC_INSTRUCTIONS\}\}/g, 
            context.ROLE_SPECIFIC_INSTRUCTIONS || "");

        return processed;
    }

    /**
     * Build complete agent context from swarm state
     */
    async buildAgentContext(
        agent: SwarmAgent,
        swarmState: SwarmState,
        swarmConfig: SwarmConfig,
        currentEvent?: unknown,
    ): Promise<string> {
        const template = await this.loadTemplate();

        // Get role-specific instructions
        const roleInstructions = this.getRoleInstructions(agent.role);

        // Build swarm state string
        const swarmStateString = this.buildSwarmStateString(swarmState, swarmConfig);

        // Get tool schemas for this agent's role
        const toolSchemas = this.getToolSchemasForRole(agent.role);

        // Process the template
        const processedPrompt = this.processTemplate(template, {
            ROLE: agent.role,
            GOAL: swarmConfig.goal || "Follow instructions and collaborate with the team",
            MEMBER_COUNT_LABEL: `${swarmState.team?.agents.length || 1} member(s)`,
            SWARM_STATE: swarmStateString,
            TOOL_SCHEMAS: JSON.stringify(toolSchemas, null, 2),
            BOT_ID: agent.id,
            ROLE_SPECIFIC_INSTRUCTIONS: roleInstructions,
        });

        // Add current event context if present
        if (currentEvent) {
            return processedPrompt + `\n\n# CURRENT EVENT\n${JSON.stringify(currentEvent, null, 2)}`;
        }

        return processedPrompt;
    }

    /**
     * Get role-specific instructions
     */
    private getRoleInstructions(role: string): string {
        // The recruitment rule from the original prompt
        const RECRUITMENT_RULE = `## Recruitment rule:
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

        switch (role.toLowerCase()) {
            case "leader":
            case "coordinator":
            case "delegator":
                return RECRUITMENT_RULE;
            
            case "analyzer":
                return "Focus on data analysis, pattern recognition, and generating insights. Validate results before reporting.";
            
            case "executor":
                return "Execute assigned tasks efficiently. Report progress and errors promptly. Stay within resource limits.";
            
            case "monitor":
                return "Monitor system health, performance, and compliance. Alert on anomalies and threshold violations.";
            
            case "validator":
                return "Validate outputs for quality, accuracy, and compliance. Check for bias and security issues.";
            
            default:
                return "Perform tasks according to your role and the overall goal.";
        }
    }

    /**
     * Build swarm state string for prompt
     */
    private buildSwarmStateString(state: SwarmState, config: SwarmConfig): string {
        const sections: string[] = [];

        // Goal and basic info
        sections.push(`GOAL: ${config.goal || "No specific goal set"}`);
        sections.push(`SWARM_LEADER: ${state.currentLeader || "No leader assigned"}`);
        
        // Team composition
        if (state.team && state.team.agents.length > 0) {
            const teamInfo = state.team.agents.map(a => 
                `  - ${a.name} (${a.role}): ${a.status}`
            ).join("\n");
            sections.push(`TEAM:\n${teamInfo}`);
        }

        // Subtasks
        if (state.subtasks && state.subtasks.length > 0) {
            const subtaskInfo = state.subtasks.map(t => 
                `  - [${t.id}] ${t.description} (${t.status})${t.assignedTo ? ` -> ${t.assignedTo}` : ""}`
            ).join("\n");
            sections.push(`SUBTASKS:\n${subtaskInfo}`);
        }

        // Resources
        if (state.resources) {
            sections.push(`RESOURCES:
  - Credits: ${state.resources.consumed.credits}/${state.resources.allocated.maxCredits}
  - Time: ${state.resources.consumed.time}/${state.resources.allocated.maxTime}ms
  - Tokens: ${state.resources.consumed.tokens}/${state.resources.allocated.maxTokens}`);
        }

        // Shared knowledge/blackboard
        if (state.blackboard && Object.keys(state.blackboard).length > 0) {
            sections.push(`BLACKBOARD:\n${JSON.stringify(state.blackboard, null, 2)}`);
        }

        return sections.join("\n\n");
    }

    /**
     * Get tool schemas based on agent role
     */
    private getToolSchemasForRole(role: string): unknown[] {
        const builtInTools = this.toolRegistry.getBuiltInDefinitions();
        const swarmTools = this.toolRegistry.getSwarmToolDefinitions();
        
        // All agents get swarm tools
        const tools = [...swarmTools];

        // Role-specific tool access
        switch (role.toLowerCase()) {
            case "leader":
            case "coordinator":
                // Leaders get all tools
                tools.push(...builtInTools);
                break;
            
            case "executor":
                // Executors get routine running and resource management
                tools.push(
                    ...builtInTools.filter(t => 
                        t.name === "run_routine" || 
                        t.name === "resource_manage"
                    )
                );
                break;
            
            case "analyzer":
            case "monitor":
            case "validator":
                // Analyzers get resource finding but not modification
                tools.push(
                    ...builtInTools.filter(t => 
                        t.name === "resource_manage" || 
                        t.name === "send_message"
                    )
                );
                break;
            
            default:
                // Other roles get basic communication
                tools.push(
                    ...builtInTools.filter(t => t.name === "send_message")
                );
        }

        return tools;
    }

    /**
     * Fallback template if file can't be loaded
     */
    private getFallbackTemplate(): string {
        return `# SWARM {{ROLE | upper}} MISSION CONTRACT

You are the {{ROLE}} of an autonomous agent swarm.

GOAL: {{GOAL}}
TIME: {{DISPLAY_DATE}}

# SWARM STATE
{{SWARM_STATE}}

# AVAILABLE TOOLS
{{TOOL_SCHEMAS}}

# INSTRUCTIONS
{{ROLE_SPECIFIC_INSTRUCTIONS}}

Make decisions based on your role, the current state, and the overall goal.
Use the available tools to accomplish tasks and coordinate with other agents.`;
    }
}