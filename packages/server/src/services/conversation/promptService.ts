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

import type { BotParticipant, ChatConfigObject, ModelType, SessionUser, TeamConfigObject } from "@vrooli/shared";
import { SECONDS_1_MS, StandardVersionConfig, getTranslation, validatePK, valueFromDot, type AgentSpec, type DataSensitivityConfig, type ResourceSubType } from "@vrooli/shared";
import * as fs from "fs/promises";
import * as path from "path";
import { fileURLToPath } from "url";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
// TODO: Import SwarmStateAccessor when available
// import type { SwarmStateAccessor } from "../execution/shared/SwarmStateAccessor.js";
import type { ToolRegistry } from "../mcp/registry.js";
import {
    DEFAULT_MEMBER_COUNT_LABEL,
    DEFAULT_TEMPLATE_FALLBACK,
    LEADERSHIP_ROLES,
    MAX_STRING_PREVIEW_LENGTH,
    RECRUITMENT_RULE_PROMPT,
    TEAM_BASED_MEMBER_COUNT_LABEL,
} from "./promptConstants.js";

// Time constants
const MINUTES_5 = 5;
const SECONDS_PER_MINUTE = 60;
const PROMPT_PREFIX_LENGTH = 7;

/**
 * Context information needed for prompt generation
 */
export interface PromptContext {
    goal: string;
    bot: BotParticipant;
    convoConfig: ChatConfigObject;
    teamConfig?: TeamConfigObject;
    toolRegistry?: ToolRegistry;
    /** User ID for security validation */
    userId?: string;
    /** Swarm ID for security context */
    swarmId?: string;
    /** Previous execution steps for variable resolution */
    previousSteps?: unknown;
    /** Input data for variable resolution */
    input?: unknown;
    /** SwarmStateAccessor for safe swarm data access */
    swarmStateAccessor?: any; // TODO: Type properly when SwarmStateAccessor is available
}

/**
 * Template variable replacements
 */
export interface TemplateVariables {
    // Special computed variables that can't be handled by dot notation
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
 * Agent prompt resolution result
 */
interface AgentPromptResolution {
    content: string;
    mode: "supplement" | "replace";
    variables: Record<string, string>;
}

/**
 * Safety event for sensitive data access
 */
interface _SafetyEvent {
    type: string;
    timestamp: number;
    data: {
        action: string;
        path: string;
        sensitivity: string;
        context: {
            agentId?: string;
            userId?: string;
            swarmId?: string;
        };
    };
}

/**
 * Safety violation error
 */
class _SafetyViolationError extends Error {
    constructor(message: string, public sensitivityType: string) {
        super(message);
        this.name = "SafetyViolationError";
    }
}

// Default configuration
const DEFAULT_CONFIG: Required<PromptServiceConfig> = {
    templateDirectory: "",  // Will be set dynamically
    defaultTemplate: "default.txt",
    enableTemplateCache: true,
    maxStringPreviewLength: MAX_STRING_PREVIEW_LENGTH,
    userPromptCacheTTL: MINUTES_5 * SECONDS_PER_MINUTE * SECONDS_1_MS,
};

/**
 * PromptService - Handles sophisticated prompt generation with templates
 * 
 * This service replaces the basic string concatenation in ContextBuilder
 * with the advanced template system from the deprecated responseEngine.ts
 */
export class PromptService {
    // Static template cache
    private static readonly templateCache = new Map<string, string | CacheEntry>();

    /**
     * 🎯 Main Entry Point: Build sophisticated system message
     * 
     * INPUT: PromptContext (bot, config, goal, etc.)
     * OUTPUT: Fully processed system message string
     * 
     * Enhanced to support agent-specific prompts that replace or supplement the base template
     * Now supports direct prompt content from bot configuration and SwarmStateAccessor integration
     */
    static async buildSystemMessage(
        context: PromptContext,
        options?: {
            templateIdentifier?: string;
            userData?: SessionUser | null;
            config?: PromptServiceConfig;
            directPromptContent?: string;
        },
    ): Promise<string> {
        const config = this.getConfig(options?.config);

        // 1. Check for direct prompt content first (highest priority)
        if (options?.directPromptContent) {
            logger.debug("[PromptService] Using direct prompt content", {
                botId: context.bot.id,
                contentLength: options.directPromptContent.length,
            });

            // Prepare template variables with SwarmStateAccessor integration
            const variables = await this.prepareTemplateVariables(
                context,
                undefined,
                config,
            );

            // Process direct content with variables
            const processedPrompt = this.processTemplate(options.directPromptContent, variables);
            await this.validatePromptSafety(processedPrompt, context);
            return processedPrompt.trim();
        }

        // 2. Check for agent-specific prompt configuration
        const agentSpec = context.bot.config?.agentSpec as AgentSpec | undefined;
        if (agentSpec?.prompt?.source === "direct" && agentSpec.prompt.content) {
            logger.debug("[PromptService] Using agent-specific direct prompt", {
                botId: context.bot.id,
                mode: agentSpec.prompt.mode,
                contentLength: agentSpec.prompt.content.length,
            });

            // Prepare template variables with custom mappings
            const variables = await this.prepareTemplateVariables(
                context,
                agentSpec.prompt.variables,
                config,
            );

            // Process agent prompt directly
            const processedPrompt = this.processTemplate(agentSpec.prompt.content, variables);
            await this.validatePromptSafety(processedPrompt, context);
            return processedPrompt.trim();
        }

        // 3. Fall back to template-based approach
        const templateName = options?.templateIdentifier || config.defaultTemplate;

        // Load base template
        const baseTemplate = await this.loadTemplate(templateName, options?.userData, config);

        // Resolve agent-specific prompt if available (resource-based or supplement mode)
        const agentPrompt = await this.resolveAgentPrompt(context, agentSpec);

        // Prepare template variables with safety checks
        const variables = await this.prepareTemplateVariables(
            context,
            agentPrompt?.variables,
            config,
        );

        // Process base template
        let systemMessage = this.processTemplate(baseTemplate, variables);

        // Apply agent prompt based on mode
        if (agentPrompt) {
            if (agentPrompt.mode === "replace") {
                systemMessage = this.processTemplate(
                    agentPrompt.content,
                    variables,
                );
            } else {
                // Supplement mode
                systemMessage = this.composePrompts(
                    systemMessage,
                    agentPrompt.content,
                    variables,
                );
            }
        }

        // Final safety validation
        await this.validatePromptSafety(systemMessage, context);

        return systemMessage.trim();
    }

    /**
     * 📁 Load Template from File System or Database
     * 
     * Loads prompt templates from external files or user-generated prompts
     * Supports both file-based templates and database prompts
     */
    static async loadTemplate(
        identifier: string,
        userData?: SessionUser | null,
        config?: PromptServiceConfig,
    ): Promise<string> {
        // Check if this is a user prompt
        if (identifier.startsWith("prompt:")) {
            const promptId = identifier.substring(PROMPT_PREFIX_LENGTH);
            const processed = await this.loadUserPrompt(promptId, userData, config);
            return processed.template;
        }

        // Otherwise, load from file system
        return this.loadFileTemplate(identifier, config);
    }

    /**
     * Load template from file system
     */
    private static async loadFileTemplate(
        templateName: string,
        config?: PromptServiceConfig,
    ): Promise<string> {
        const cfg = this.getConfig(config);
        const fileName = templateName || cfg.defaultTemplate;

        // Check cache first
        if (cfg.enableTemplateCache && this.templateCache.has(fileName)) {
            const cached = this.templateCache.get(fileName);
            if (typeof cached === "string") {
                return cached;
            }
        }

        const templatePath = path.join(cfg.templateDirectory, fileName);

        try {
            logger.debug(`[PromptService] Loading template from: ${templatePath}`);
            const template = await fs.readFile(templatePath, "utf-8");

            // Cache the template
            if (cfg.enableTemplateCache) {
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
    private static async loadUserPrompt(
        promptVersionId: string,
        userData?: SessionUser | null,
        config?: PromptServiceConfig,
    ): Promise<ProcessedPrompt> {
        const cfg = this.getConfig(config);

        // 1. Check cache first
        const cacheKey = `prompt:${promptVersionId}`;
        if (cfg.enableTemplateCache && this.templateCache.has(cacheKey)) {
            const cached = this.templateCache.get(cacheKey) as CacheEntry;
            if (cached && this.isUserPromptCacheValid(cached, cfg)) {
                return cached.content as unknown as ProcessedPrompt;
            }
        }

        // 2. Build query using existing patterns

        // 3. Query database with correct identifier and relationship structure
        // Use id if promptVersionId is a valid primary key, otherwise use publicId
        const isValidPK = validatePK(promptVersionId);
        const whereClause = isValidPK
            ? { id: BigInt(promptVersionId), resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false }
            : { publicId: promptVersionId, resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false };

        const promptVersion = await DbProvider.get().resource_version.findUnique({
            where: whereClause,
            select: {
                id: true,
                config: true,
                isDeleted: true,
                isPrivate: true,
                resourceSubType: true,
                translations: {
                    select: {
                        id: true,
                        language: true,
                        name: true,
                        description: true,
                    },
                },
                root: {
                    select: {
                        id: true,
                        isDeleted: true,
                        isPrivate: true,
                        ownedByUserId: true,
                        ownedByTeamId: true,
                        ownedByTeam: userData ? {
                            select: {
                                id: true,
                                members: {
                                    select: {
                                        userId: true,
                                        isAdmin: true,
                                        permissions: true,
                                    },
                                },
                            },
                        } : false,
                        ownedByUser: {
                            select: {
                                id: true,
                            },
                        },
                    },
                },
            },
        });

        if (!promptVersion) {
            throw new CustomError("0022", "NotFound", { objectType: "Prompt" });
        }

        // 4. Use existing permission infrastructure with actual database ID
        const actualId = promptVersion.id.toString();
        const authDataById = await getAuthenticatedData(
            { ResourceVersion: [actualId] } as { [key in ModelType]?: string[] },
            userData ?? null,
        );

        // Check read permissions using existing system
        await permissionsCheck(
            authDataById,
            { Read: [actualId] },
            {},
            userData ?? null,
        );

        // 5. Process and cache the prompt
        const processed = this.processUserPrompt(promptVersion as any, userData?.languages?.[0] || "en");

        // Cache with TTL
        if (cfg.enableTemplateCache) {
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
    private static isUserPromptCacheValid(entry: CacheEntry, config: Required<PromptServiceConfig>): boolean {
        if (!config.userPromptCacheTTL) return true;
        return Date.now() - entry.cachedAt < config.userPromptCacheTTL;
    }

    /**
     * Process user prompt from database into usable format
     */
    private static processUserPrompt(
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
            JSON.parse(config.schema as string) : null;

        // Get translated metadata
        const translation = getTranslation(promptVersion, [language]);

        return {
            template,
            inputSchema,
            metadata: {
                name: translation.name || "",
                description: translation.description || "",
                version: promptVersion.versionLabel || "",
            },
            processingRules: config.props as Record<string, unknown> | undefined,
        };
    }

    /**
     * Extract template string from config using proper StandardVersionConfig parser
     */
    private static extractTemplateFromConfig(config: Record<string, unknown>): string {
        try {
            // Use StandardVersionConfig constructor directly to avoid type issues
            const parsedConfig = new StandardVersionConfig({
                config: {
                    __version: (config.__version as string) || "1.0",
                    ...config,
                },
                resourceSubType: "StandardPrompt" as ResourceSubType,
            });

            // Check props.template first (most direct)
            if (parsedConfig.props?.template && typeof parsedConfig.props.template === "string") {
                return parsedConfig.props.template;
            }

            // Check props.content second
            if (parsedConfig.props?.content && typeof parsedConfig.props.content === "string") {
                return parsedConfig.props.content;
            }

            // Check schema field - could contain template as JSON or raw text
            if (parsedConfig.schema && typeof parsedConfig.schema === "string") {
                try {
                    const parsed = JSON.parse(parsedConfig.schema) as Record<string, unknown>;
                    if (parsed.template && typeof parsed.template === "string") {
                        return parsed.template;
                    }
                    if (parsed.content && typeof parsed.content === "string") {
                        return parsed.content;
                    }
                } catch {
                    // If schema is not JSON, treat it as the raw template
                    return parsedConfig.schema;
                }
            }

            logger.warn("[PromptService] No template found in StandardPrompt config", {
                hasProps: !!parsedConfig.props,
                hasSchema: !!parsedConfig.schema,
                propsKeys: parsedConfig.props ? Object.keys(parsedConfig.props) : [],
            });

        } catch (error) {
            logger.error("[PromptService] Failed to parse StandardPrompt config", { error });
        }

        return DEFAULT_TEMPLATE_FALLBACK;
    }

    /**
     * Resolve agent-specific prompt configuration
     */
    private static async resolveAgentPrompt(
        context: PromptContext,
        agentSpec?: AgentSpec,
    ): Promise<AgentPromptResolution | null> {
        if (!agentSpec?.prompt) return null;

        if (agentSpec.prompt.source === "direct") {
            if (!agentSpec.prompt.content) {
                throw new Error("Direct prompt source requires content field");
            }
            return {
                content: agentSpec.prompt.content,
                mode: agentSpec.prompt.mode,
                variables: agentSpec.prompt.variables || {},
            };
        }

        // Resource-based prompt
        if (!agentSpec.prompt.resourceId) {
            throw new Error("Resource prompt source requires resourceId field");
        }
        const resource = await this.loadPromptResource(
            agentSpec.prompt.resourceId,
            context,
        );

        return {
            content: resource.content,
            mode: agentSpec.prompt.mode,
            variables: agentSpec.prompt.variables || {},
        };
    }

    /**
     * Load prompt from resource
     */
    private static async loadPromptResource(
        resourceId: string,
        _context: PromptContext,
    ): Promise<{ content: string }> {
        try {
            // Load prompt resource from database using existing patterns
            // Use id if resourceId is a valid primary key, otherwise use publicId
            const isValidPK = validatePK(resourceId);
            const whereClause = isValidPK
                ? { id: BigInt(resourceId), resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false }
                : { publicId: resourceId, resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false };

            const promptVersion = await DbProvider.get().resource_version.findUnique({
                where: whereClause,
                select: {
                    config: true,
                },
            });

            if (!promptVersion) {
                throw new CustomError("0022", "NotFound", { objectType: "PromptResource" });
            }

            const config = (promptVersion.config || {}) as Record<string, unknown>;
            const content = this.extractTemplateFromConfig(config);

            return { content };
        } catch (error) {
            logger.error("[PromptService] Failed to load prompt resource", { resourceId, error });
            return { content: DEFAULT_TEMPLATE_FALLBACK };
        }
    }

    /**
     * Prepare template variables with safety checks
     */
    private static async prepareTemplateVariables(
        context: PromptContext,
        customMappings?: Record<string, string>,
        config?: Required<PromptServiceConfig>,
    ): Promise<TemplateVariables> {
        const cfg = config || this.getConfig();

        // Build standard variables
        const variables = this.buildTemplateVariables(context, cfg);

        // Resolve custom mappings with safety checks
        if (customMappings) {
            for (const [varName, path] of Object.entries(customMappings)) {
                try {
                    const value = await this.resolveVariableWithSafety(
                        path,
                        context,
                    );
                    (variables as any)[varName] = value;
                } catch (error) {
                    logger.warn(`[PromptService] Failed to resolve variable ${varName} from path ${path}`, { error });
                    (variables as any)[varName] = "";
                }
            }
        }

        return variables;
    }

    /**
     * Resolve variable with safety checks using SwarmStateAccessor
     */
    private static async resolveVariableWithSafety(
        path: string,
        context: PromptContext,
    ): Promise<string> {
        const [scope, ...pathParts] = path.split(".");

        // Check if accessing sensitive data
        const sensitivity = this.checkDataSensitivity(path, context);
        if (sensitivity) {
            await this.handleSensitiveDataAccess(
                path,
                sensitivity,
                context,
            );
        }

        // Use SwarmStateAccessor for swarm-related data if available
        if (context.swarmStateAccessor && (scope === "swarm" || scope === "context" || scope === "config")) {
            try {
                const executionContext = {
                    agentId: context.bot.id,
                    swarmId: context.swarmId || "unknown",
                    userId: context.userId,
                    swarmConfig: {
                        secrets: context.convoConfig?.secrets,
                        ...context.convoConfig,
                    },
                };

                const value = await context.swarmStateAccessor.accessData(
                    pathParts.join("."),
                    executionContext,
                    {
                        requireValidation: true,
                        logAccess: true,
                    },
                );

                return String(value || "");
            } catch (error) {
                logger.warn(`[PromptService] SwarmStateAccessor failed for path ${path}`, { error });
                // Fall back to direct access with logging
            }
        }

        // Fall back to direct access for non-swarm data or when SwarmStateAccessor unavailable
        switch (scope) {
            case "input":
                return String(valueFromDot(context.input as any, pathParts.join(".")) || "");
            case "context":
                return String(valueFromDot(context as any, pathParts.join(".")) || "");
            case "config":
                return String(valueFromDot(context.convoConfig as any, pathParts.join(".")) || "");
            case "previous":
                return String(valueFromDot(context.previousSteps as any, pathParts.join(".")) || "");
            case "swarm":
                // If no SwarmStateAccessor, try direct access to convoConfig
                return String(valueFromDot(context.convoConfig as any, pathParts.join(".")) || "");
            default:
                throw new Error(`Unknown variable scope: ${scope}`);
        }
    }

    /**
     * Check if data path is sensitive
     */
    private static checkDataSensitivity(
        path: string,
        context: PromptContext,
    ): DataSensitivityConfig | null {
        const secrets = context.convoConfig?.secrets;
        if (!secrets) return null;

        // Check if path matches any secret pattern
        for (const [pattern, sensitivity] of Object.entries(secrets)) {
            if (this.pathMatchesPattern(path, pattern)) {
                return sensitivity;
            }
        }

        return null;
    }

    /**
     * Check if path matches pattern (simple glob-like matching)
     */
    private static pathMatchesPattern(path: string, pattern: string): boolean {
        // Convert glob pattern to regex
        const regexPattern = pattern
            .replace(/\*/g, ".*")
            .replace(/\?/g, ".")
            .replace(/\./g, "\\.");

        const regex = new RegExp(`^${regexPattern}$`);
        return regex.test(path);
    }

    /**
     * Handle sensitive data access
     */
    private static async handleSensitiveDataAccess(
        path: string,
        sensitivity: DataSensitivityConfig,
        context: PromptContext,
    ): Promise<void> {
        // For now, just log the access - in full implementation this would
        // emit safety events to the event bus
        if (sensitivity.accessLog) {
            logger.info("[PromptService] Accessing sensitive data", {
                path,
                sensitivity: sensitivity.type,
                agentId: context.bot.id,
                userId: context.userId,
                swarmId: context.swarmId,
            });
        }

        // In the full implementation, this would check sensitivity.requireConfirmation
        // and emit pre-action safety events
    }

    /**
     * Compose prompts for supplement mode
     */
    private static composePrompts(
        basePrompt: string,
        agentPrompt: string,
        variables: TemplateVariables,
    ): string {
        const processedAgentPrompt = this.processTemplate(agentPrompt, variables);

        return `${basePrompt}\n\n## Agent-Specific Instructions\n${processedAgentPrompt}`;
    }

    /**
     * Validate prompt safety (placeholder for future implementation)
     */
    private static async validatePromptSafety(
        prompt: string,
        context: PromptContext,
    ): Promise<void> {
        // Placeholder for prompt injection detection and other safety checks
        // In full implementation, this would validate the final prompt for safety
        logger.debug("[PromptService] Validating prompt safety", {
            promptLength: prompt.length,
            agentId: context.bot.id,
        });
    }

    /**
     * 🔄 Process Template Variables
     * 
     * Replaces template variables with actual values
     * Migrated from _processPromptTemplate() in responseEngine.ts
     */
    static processTemplate(template: string, variables: TemplateVariables): string {
        let processedPrompt = template;

        // Replace special computed variables that can't be handled by dot notation
        processedPrompt = processedPrompt.replace(/{{MEMBER_COUNT_LABEL}}/g, variables.memberCountLabel);
        processedPrompt = processedPrompt.replace(/{{ISO_EPOCH_SECONDS}}/g, variables.isoEpochSeconds);
        processedPrompt = processedPrompt.replace(/{{DISPLAY_DATE}}/g, variables.displayDate);
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
    private static buildTemplateVariables(
        context: PromptContext,
        config: Required<PromptServiceConfig>,
    ): TemplateVariables {
        const { bot, convoConfig, teamConfig, toolRegistry } = context;
        const botRole = bot.role || "leader"; // Default to leader if no role

        // Determine member count label
        let memberCountLabel = DEFAULT_MEMBER_COUNT_LABEL;
        if (convoConfig.teamId) {
            memberCountLabel = TEAM_BASED_MEMBER_COUNT_LABEL;
        }

        // Get role-specific instructions
        const roleSpecificInstructions = this.getRoleSpecificInstructions(botRole);

        // Build swarm state string
        const swarmState = this.formatSwarmState(convoConfig, teamConfig, config);

        // Build tool schemas string
        const toolSchemas = this.buildToolSchemasString(toolRegistry);

        return {
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
    private static getRoleSpecificInstructions(botRole: string): string {
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
    static formatSwarmState(
        convoConfig: ChatConfigObject | undefined,
        teamConfig?: TeamConfigObject,
        config?: PromptServiceConfig,
    ): string {
        const cfg = this.getConfig(config);
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
                formattedSwarmStateParts.push(`- Team Configuration:\n${this.truncateForPrompt(teamInfo, cfg.maxStringPreviewLength)}\n`);
            } else {
                formattedSwarmStateParts.push(`- Team ID:\n${this.truncateForPrompt(teamId, cfg.maxStringPreviewLength)}\n`);
            }

            formattedSwarmStateParts.push(`- Swarm Leader:\n${this.truncateForPrompt(swarmLeader, cfg.maxStringPreviewLength)}\n`);

            // Subtasks with counts
            const activeSubtasksCount = subtasks.filter(st =>
                typeof st === "object" && st !== null && (st.status === "todo" || st.status === "in_progress"),
            ).length;
            const completedSubtasksCount = subtasks.filter(st =>
                typeof st === "object" && st !== null && st.status === "done",
            ).length;
            formattedSwarmStateParts.push(`- Subtasks (active: ${activeSubtasksCount}, completed: ${completedSubtasksCount}):\n${this.truncateForPrompt(subtasks, cfg.maxStringPreviewLength)}\n`);

            // Other swarm state components
            formattedSwarmStateParts.push(`- Subtask Leaders:\n${this.truncateForPrompt(subtaskLeaders, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Event Subscriptions:\n${this.truncateForPrompt(eventSubscriptions, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Blackboard:\n${this.truncateForPrompt(blackboard, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Resources:\n${this.truncateForPrompt(resources, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Records:\n${this.truncateForPrompt(records, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Stats:\n${this.truncateForPrompt(stats, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Limits:\n${this.truncateForPrompt(limits, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Pending Tool Calls:\n${this.truncateForPrompt(pendingToolCalls, cfg.maxStringPreviewLength)}\n`);

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
    private static buildToolSchemasString(toolRegistry?: ToolRegistry): string {
        if (!toolRegistry) {
            return "No tools available for this role.";
        }

        try {
            const builtInToolSchemas = toolRegistry.getBuiltInDefinitions();
            const swarmToolSchemas = toolRegistry.getSwarmToolDefinitions();
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
    private static truncateForPrompt(value: unknown, maxLength: number): string {
        const limit = maxLength;

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
    private static getDefaultTemplateDirectory(): string {
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        return path.join(__dirname, "templates");
    }

    /**
     * Get merged configuration
     */
    private static getConfig(config?: PromptServiceConfig): Required<PromptServiceConfig> {
        return {
            ...DEFAULT_CONFIG,
            templateDirectory: config?.templateDirectory || this.getDefaultTemplateDirectory(),
            ...config,
        };
    }

    /**
     * 🧹 Clear Template Cache
     * 
     * Clears the template cache (useful for development/testing)
     */
    static clearTemplateCache(): void {
        this.templateCache.clear();
        logger.debug("[PromptService] Template cache cleared");
    }

    /**
     * 📊 Get Cache Statistics
     * 
     * Returns information about template cache usage
     */
    static getCacheStats(): { size: number; enabled: boolean } {
        return {
            size: this.templateCache.size,
            enabled: DEFAULT_CONFIG.enableTemplateCache,
        };
    }
}
