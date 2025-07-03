/**
 * PromptService - Advanced Template-Based Prompt Generation
 * 
 * This service provides sophisticated prompt generation capabilities extracted
 * from the deprecated responseEngine.ts, including template processing, swarm
 * state formatting, and role-specific instructions.
 * 
 * Key Features:
 * - Template-based prompting with variable substitution
 * - Rich swarm state formatting
 * - Role-specific instructions
 * - Single default template with database prompt support
 * - Smart content truncation
 */

import type { BotParticipant, ChatConfigObject, TeamConfigObject, SessionUser, ModelType } from "@vrooli/shared";
import { SECONDS_1_MS, getTranslation, type ResourceSubType } from "@vrooli/shared";
import * as fs from "fs/promises";
import * as path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events/logger.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import type { ToolRegistry } from "../mcp/registry.js";
import {
    DEFAULT_TEMPLATE_FALLBACK,
    LEADERSHIP_ROLES,
    MAX_STRING_PREVIEW_LENGTH,
    RECRUITMENT_RULE_PROMPT,
    DEFAULT_MEMBER_COUNT_LABEL,
    TEAM_BASED_MEMBER_COUNT_LABEL,
} from "./promptConstants.js";

// Time constants
const MINUTES_5 = 5;
const SECONDS_PER_MINUTE = 60;

/**
 * Context information needed for prompt generation
 */
export interface PromptContext {
    goal: string;
    bot: BotParticipant;
    convoConfig: ChatConfigObject;
    teamConfig?: TeamConfigObject;
    toolRegistry?: ToolRegistry;
    /** User-provided inputs for dynamic prompts */
    userInputs?: Record<string, unknown>;
}

/**
 * Template variable replacements
 */
export interface TemplateVariables {
    goal: string;
    role: string;
    botId: string;
    memberCountLabel: string;
    isoEpochSeconds: string;
    displayDate: string;
    roleSpecificInstructions: string;
    swarmState: string;
    toolSchemas: string;
}

/**
 * Configuration options for PromptService
 */
export interface PromptServiceConfig {
    templateDirectory?: string;
    defaultTemplate?: string;
    enableTemplateCache?: boolean;
    maxStringPreviewLength?: number;
    /** Cache TTL for user prompts in milliseconds */
    userPromptCacheTTL?: number;
}

/**
 * Cache entry for templates
 */
interface CacheEntry {
    content: string;
    metadata?: {
        name: string;
        description: string;
        version: string;
    };
    cachedAt: number;
}

/**
 * Processed prompt data
 */
interface ProcessedPrompt {
    template: string;
    inputSchema?: Record<string, unknown>;
    metadata?: {
        name: string;
        description: string;
        version: string;
    };
    processingRules?: Record<string, unknown>;
}

/**
 * PromptService - Handles sophisticated prompt generation with templates
 * 
 * This service replaces the basic string concatenation in ContextBuilder
 * with the advanced template system from the deprecated responseEngine.ts
 */
export class PromptService {
    private readonly config: Required<PromptServiceConfig>;
    private readonly templateCache = new Map<string, string | CacheEntry>();

    constructor(
        private readonly toolRegistry?: ToolRegistry,
        config: PromptServiceConfig = {},
    ) {
        // Set default configuration
        this.config = {
            templateDirectory: config.templateDirectory || this.getDefaultTemplateDirectory(),
            defaultTemplate: config.defaultTemplate || "default.txt",
            enableTemplateCache: config.enableTemplateCache ?? true,
            maxStringPreviewLength: config.maxStringPreviewLength || MAX_STRING_PREVIEW_LENGTH,
            userPromptCacheTTL: config.userPromptCacheTTL || MINUTES_5 * SECONDS_PER_MINUTE * SECONDS_1_MS, // 5 minutes default
        };

        logger.info("[PromptService] Initialized with template-based prompt generation", {
            templateDirectory: this.config.templateDirectory,
            enableCache: this.config.enableTemplateCache,
        });
    }

    /**
     * 🎯 Main Entry Point: Build sophisticated system message
     * 
     * INPUT: PromptContext (bot, config, goal, etc.)
     * OUTPUT: Fully processed system message string
     * 
     * This replaces the basic buildSystemMessage in ContextBuilder
     */
    async buildSystemMessage(
        context: PromptContext,
        options?: {
            templateIdentifier?: string;
            userData?: SessionUser | null;
        },
    ): Promise<string> {
        const identifier = options?.templateIdentifier || this.config.defaultTemplate;
        
        // Load template (handles both file and user prompts)
        const template = await this.loadTemplate(identifier, options?.userData);
        
        // Build standard variables
        const variables = this.buildTemplateVariables(context);
        
        // Process template
        let processedPrompt = this.processTemplate(template, variables);
        
        // If user inputs provided and this is a user prompt, apply them
        if (identifier.startsWith("prompt:") && context.userInputs) {
            processedPrompt = this.applyUserInputs(processedPrompt, context.userInputs);
        }
        
        return processedPrompt.trim();
    }

    /**
     * 📁 Load Template from File System or Database
     * 
     * Loads prompt templates from external files or user-generated prompts
     * Supports both file-based templates and database prompts
     */
    async loadTemplate(identifier: string, userData?: SessionUser | null): Promise<string> {
        // Check if this is a user prompt
        if (identifier.startsWith("prompt:")) {
            const PROMPT_PREFIX_LENGTH = 7;
            const promptId = identifier.substring(PROMPT_PREFIX_LENGTH);
            const processed = await this.loadUserPrompt(promptId, userData);
            return processed.template;
        }
        
        // Otherwise, load from file system
        return this.loadFileTemplate(identifier);
    }

    /**
     * Load template from file system
     */
    private async loadFileTemplate(templateName: string): Promise<string> {
        const fileName = templateName || this.config.defaultTemplate;
        
        // Check cache first
        if (this.config.enableTemplateCache && this.templateCache.has(fileName)) {
            const cached = this.templateCache.get(fileName);
            if (typeof cached === "string") {
                return cached;
            }
        }

        const templatePath = path.join(this.config.templateDirectory, fileName);

        try {
            logger.debug(`[PromptService] Loading template from: ${templatePath}`);
            const template = await fs.readFile(templatePath, "utf-8");
            
            // Cache the template
            if (this.config.enableTemplateCache) {
                this.templateCache.set(fileName, template);
            }
            
            return template;
        } catch (error) {
            logger.error(`[PromptService] Failed to load template from ${templatePath}`, { error });
            return DEFAULT_TEMPLATE_FALLBACK;
        }
    }

    /**
     * Load user-generated prompt from database
     */
    private async loadUserPrompt(
        promptVersionId: string,
        userData?: SessionUser | null,
    ): Promise<ProcessedPrompt> {
        // 1. Check cache first
        const cacheKey = `prompt:${promptVersionId}`;
        if (this.config.enableTemplateCache && this.templateCache.has(cacheKey)) {
            const cached = this.templateCache.get(cacheKey) as CacheEntry;
            if (cached && this.isUserPromptCacheValid(cached)) {
                return cached.content as unknown as ProcessedPrompt;
            }
        }

        // 2. Build query using existing patterns
        
        // Build where clause
        const where = {
            id: promptVersionId,
            resourceSubType: "StandardPrompt" as ResourceSubType,
            isDeleted: false,
        };
        
        // Use partial info for select
        const partialInfo = {
            id: true,
            config: true,
            isPrivate: true,
            resourceSubType: true,
            translations: {
                id: true,
                language: true,
                name: true,
                description: true,
            },
            root: {
                id: true,
                isPrivate: true,
                owner: {
                    __typename: true,
                    id: true,
                    // For team ownership checks
                    members: userData ? {
                        id: true,
                        userId: true,
                        roles: {
                            id: true,
                            permissions: true,
                        },
                    } : false,
                },
            },
        };
        
        // 3. Query database
        const promptVersion = await DbProvider.get().resource_version.findUnique({
            where,
            ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo),
        });
        
        if (!promptVersion) {
            throw new CustomError("0022", "NotFound", { objectType: "Prompt" });
        }
        
        // 4. Use existing permission infrastructure
        const authDataById = await getAuthenticatedData(
            { ResourceVersion: [promptVersionId] } as { [key in ModelType]?: string[] }, 
            userData ?? null,
        );
        
        // Check read permissions using existing system
        await permissionsCheck(
            authDataById, 
            { Read: [promptVersionId] }, 
            {}, 
            userData,
        );
        
        // 5. Process and cache the prompt
        const processed = this.processUserPrompt(promptVersion, userData?.languages?.[0] || "en");
        
        // Cache with TTL
        if (this.config.enableTemplateCache) {
            this.templateCache.set(cacheKey, {
                content: processed as unknown as string,
                cachedAt: Date.now(),
            });
        }
        
        return processed;
    }

    /**
     * Check if user prompt cache is still valid
     */
    private isUserPromptCacheValid(entry: CacheEntry): boolean {
        if (!this.config.userPromptCacheTTL) return true;
        return Date.now() - entry.cachedAt < this.config.userPromptCacheTTL;
    }

    /**
     * Process user prompt from database into usable format
     */
    private processUserPrompt(
        promptVersion: {
            config?: Record<string, unknown>;
            translations?: Array<{ language: string; name?: string; description?: string }>;
            versionLabel?: string;
            [key: string]: unknown;
        },
        language: string,
    ): ProcessedPrompt {
        const config = (promptVersion.config || {}) as Record<string, unknown>;
        
        // Extract template from config
        const template = this.extractTemplateFromConfig(config);
        
        // Parse input schema if present
        const inputSchema = config.schema ? 
            JSON.parse(config.schema) : null;
        
        // Get translated metadata
        const translation = getTranslation(promptVersion, [language]);
        
        return {
            template,
            inputSchema,
            metadata: {
                name: translation.name || "",
                description: translation.description || "",
                version: promptVersion.versionLabel,
            },
            processingRules: config.props as Record<string, unknown> | undefined,
        };
    }

    /**
     * Extract template string from config
     */
    private extractTemplateFromConfig(config: Record<string, unknown>): string {
        // Template could be in various places depending on how the prompt was created
        const props = config.props as Record<string, unknown> | undefined;
        if (props?.template && typeof props.template === "string") {
            return props.template;
        }
        if (props?.content && typeof props.content === "string") {
            return props.content;
        }
        if (config.schema && typeof config.schema === "string") {
            try {
                const parsed = JSON.parse(config.schema) as Record<string, unknown>;
                if (parsed.template && typeof parsed.template === "string") return parsed.template;
                if (parsed.content && typeof parsed.content === "string") return parsed.content;
            } catch {
                // If schema is not JSON, treat it as the template
                return config.schema;
            }
        }
        return DEFAULT_TEMPLATE_FALLBACK;
    }

    /**
     * Apply user inputs to template
     */
    private applyUserInputs(
        template: string,
        inputs: Record<string, unknown>,
    ): string {
        let processed = template;
        
        // Replace input placeholders with actual values
        for (const [key, value] of Object.entries(inputs)) {
            // Escape special regex characters in the key
            const escapedKey = key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
            // Support multiple placeholder formats
            processed = processed.replace(new RegExp(`{{\\s*${escapedKey}\\s*}}`, "g"), String(value));
            processed = processed.replace(new RegExp(`{${escapedKey}}`, "g"), String(value));
            processed = processed.replace(new RegExp(`\\$\\{${escapedKey}\\}`, "g"), String(value));
        }
        
        return processed;
    }

    /**
     * 🔄 Process Template Variables
     * 
     * Replaces template variables with actual values
     * Migrated from _processPromptTemplate() in responseEngine.ts
     */
    processTemplate(template: string, variables: TemplateVariables): string {
        let processedPrompt = template;

        // Replace all template variables
        processedPrompt = processedPrompt.replace(/{{GOAL}}/g, variables.goal);
        processedPrompt = processedPrompt.replace(/{{MEMBER_COUNT_LABEL}}/g, variables.memberCountLabel);
        processedPrompt = processedPrompt.replace(/{{ISO_EPOCH_SECONDS}}/g, variables.isoEpochSeconds);
        processedPrompt = processedPrompt.replace(/{{DISPLAY_DATE}}/g, variables.displayDate);
        processedPrompt = processedPrompt.replace(/{{ROLE\s*\|\s*upper}}/g, variables.role.toUpperCase());
        processedPrompt = processedPrompt.replace(/{{ROLE}}/g, variables.role);
        processedPrompt = processedPrompt.replace(/{{BOT_ID}}/g, variables.botId);
        processedPrompt = processedPrompt.replace(/{{ROLE_SPECIFIC_INSTRUCTIONS}}/g, variables.roleSpecificInstructions);
        processedPrompt = processedPrompt.replace(/{{SWARM_STATE}}/g, variables.swarmState);
        processedPrompt = processedPrompt.replace(/{{TOOL_SCHEMAS}}/g, variables.toolSchemas);

        return processedPrompt;
    }

    /**
     * 🏗️ Build Template Variables from Context
     * 
     * Constructs all template variables from the provided context
     */
    private buildTemplateVariables(context: PromptContext): TemplateVariables {
        const { goal, bot, convoConfig, teamConfig } = context;
        const botRole = bot.meta?.role || "leader"; // Default to leader if no role
        const botId = bot.id;

        // Determine member count label
        let memberCountLabel = DEFAULT_MEMBER_COUNT_LABEL;
        if (convoConfig.teamId) {
            memberCountLabel = TEAM_BASED_MEMBER_COUNT_LABEL;
        }

        // Get role-specific instructions
        const roleSpecificInstructions = this.getRoleSpecificInstructions(botRole);

        // Build swarm state string
        const swarmState = this.formatSwarmState(convoConfig, teamConfig);

        // Build tool schemas string
        const toolSchemas = this.buildToolSchemasString();

        return {
            goal,
            role: botRole,
            botId,
            memberCountLabel,
            isoEpochSeconds: Math.floor(Date.now() / SECONDS_1_MS).toString(),
            displayDate: new Date().toLocaleString(),
            roleSpecificInstructions,
            swarmState,
            toolSchemas,
        };
    }

    /**
     * 🎭 Get Role-Specific Instructions
     * 
     * Returns appropriate instructions based on bot role
     */
    private getRoleSpecificInstructions(botRole: string): string {
        if (LEADERSHIP_ROLES.includes(botRole as typeof LEADERSHIP_ROLES[number])) {
            return RECRUITMENT_RULE_PROMPT;
        }
        return "Perform tasks according to your role and the overall goal.";
    }

    /**
     * 🌊 Format Swarm State Information
     * 
     * Creates a rich formatted string of the current swarm state
     * Migrated from _buildSwarmStateString() in responseEngine.ts
     */
    formatSwarmState(convoConfig: ChatConfigObject | undefined, teamConfig?: TeamConfigObject): string {
        let swarmStateOutput = "SWARM STATE DETAILS: Not available or error formatting state.";

        if (!convoConfig) {
            return "SWARM STATE DETAILS: Configuration not available.";
        }

        try {
            const teamId = convoConfig.teamId || "No team assigned";
            const swarmLeader = convoConfig.swarmLeader || "No leader assigned";
            const subtasks = convoConfig.subtasks || [];
            const subtaskLeaders = convoConfig.subtaskLeaders || {};
            const eventSubscriptions = convoConfig.eventSubscriptions || {};
            const blackboard = convoConfig.blackboard || [];
            const resources = convoConfig.resources || [];
            const records = convoConfig.records || [];
            const stats = convoConfig.stats || {};
            const limits = convoConfig.limits || {};
            const pendingToolCalls = convoConfig.pendingToolCalls || [];

            const formattedSwarmStateParts: string[] = [];

            // Team information with organizational structure
            if (teamConfig) {
                const teamInfo = {
                    id: teamId,
                    structure: teamConfig.structure || { type: "Not specified", content: "No organizational structure defined" },
                };
                formattedSwarmStateParts.push(`- Team Configuration:\n${this.truncateForPrompt(teamInfo)}\n`);
            } else {
                formattedSwarmStateParts.push(`- Team ID:\n${this.truncateForPrompt(teamId)}\n`);
            }

            formattedSwarmStateParts.push(`- Swarm Leader:\n${this.truncateForPrompt(swarmLeader)}\n`);

            // Subtasks with counts
            const activeSubtasksCount = subtasks.filter(st => 
                typeof st === "object" && st !== null && (st.status === "todo" || st.status === "in_progress"),
            ).length;
            const completedSubtasksCount = subtasks.filter(st => 
                typeof st === "object" && st !== null && st.status === "done",
            ).length;
            formattedSwarmStateParts.push(`- Subtasks (active: ${activeSubtasksCount}, completed: ${completedSubtasksCount}):\n${this.truncateForPrompt(subtasks)}\n`);

            // Other swarm state components
            formattedSwarmStateParts.push(`- Subtask Leaders:\n${this.truncateForPrompt(subtaskLeaders)}\n`);
            formattedSwarmStateParts.push(`- Event Subscriptions:\n${this.truncateForPrompt(eventSubscriptions)}\n`);
            formattedSwarmStateParts.push(`- Blackboard:\n${this.truncateForPrompt(blackboard)}\n`);
            formattedSwarmStateParts.push(`- Resources:\n${this.truncateForPrompt(resources)}\n`);
            formattedSwarmStateParts.push(`- Records:\n${this.truncateForPrompt(records)}\n`);
            formattedSwarmStateParts.push(`- Stats:\n${this.truncateForPrompt(stats)}\n`);
            formattedSwarmStateParts.push(`- Limits:\n${this.truncateForPrompt(limits)}\n`);
            formattedSwarmStateParts.push(`- Pending Tool Calls:\n${this.truncateForPrompt(pendingToolCalls)}\n`);

            swarmStateOutput = `\nSWARM STATE DETAILS:\n${formattedSwarmStateParts.join("\n\n")}`;
        } catch (e) {
            logger.error("[PromptService] Error formatting swarm state for prompt", { error: e });
        }

        return swarmStateOutput;
    }

    /**
     * 🔧 Build Tool Schemas String
     * 
     * Formats available tool schemas for inclusion in prompts
     */
    private buildToolSchemasString(): string {
        if (!this.toolRegistry) {
            return "No tools available for this role.";
        }

        try {
            const builtInToolSchemas = this.toolRegistry.getBuiltInDefinitions();
            const swarmToolSchemas = this.toolRegistry.getSwarmToolDefinitions();
            const allToolSchemas = [...builtInToolSchemas, ...swarmToolSchemas];

            return allToolSchemas.length > 0 
                ? JSON.stringify(allToolSchemas, null, 2)
                : "No tools available for this role.";
        } catch (error) {
            logger.error("[PromptService] Error building tool schemas string", { error });
            return "Error loading tool schemas.";
        }
    }

    /**
     * ✂️ Smart Content Truncation
     * 
     * Truncates content for prompts while preserving readability
     * Migrated from _truncateStringForPrompt() in responseEngine.ts
     */
    private truncateForPrompt(value: unknown, maxLength?: number): string {
        const limit = maxLength || this.config.maxStringPreviewLength;
        
        if (value === undefined || value === null) {
            return "Not set";
        }
        
        const stringifiedValue = typeof value === "string" ? value : JSON.stringify(value, null, 2);
        
        if (stringifiedValue.length <= limit) {
            return stringifiedValue;
        }
        
        return stringifiedValue.substring(0, limit) + "...";
    }

    /**
     * 📂 Get Default Template Directory
     * 
     * Determines the default location for template files
     */
    private getDefaultTemplateDirectory(): string {
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        return path.join(__dirname, "templates");
    }

    /**
     * 🧹 Clear Template Cache
     * 
     * Clears the template cache (useful for development/testing)
     */
    clearTemplateCache(): void {
        this.templateCache.clear();
        logger.debug("[PromptService] Template cache cleared");
    }

    /**
     * 📊 Get Cache Statistics
     * 
     * Returns information about template cache usage
     */
    getCacheStats(): { size: number; enabled: boolean } {
        return {
            size: this.templateCache.size,
            enabled: this.config.enableTemplateCache,
        };
    }
}
