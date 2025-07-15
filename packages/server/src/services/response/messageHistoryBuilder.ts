/**
 * Unified Prompt Builder - Clean LLM Message Array Generation
 * 
 * This service provides a single method that ResponseService needs:
 * buildMessages() returns a complete message array ready for any LLM.
 * 
 * Message Array Format:
 * [0] = System message (built from PromptService + ResponseContext)
 * [1..n] = Conversation history (token-budgeted, Redis-cached)
 */

import type { ChatMessage, MessageConfigObject, ResponseContext } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import type { MessageState } from "../conversation/types.js";
import { ToolRegistry } from "../mcp/registry.js";
import { ChatContextCache, TokenCounter } from "./messageStore.js";
import { PromptService, type PromptContext } from "./promptService.js";
import { AIServiceRegistry } from "./registry.js";

const CONTEXT_SIZE_SAFETY_BUFFER_PERCENT = 0.05;
const MAX_HISTORY_ENTRIES = 1000;
const SYSTEM_MESSAGE_BUDGET_PERCENT = 0.25;
const RESPONSE_RESERVE_TOKENS = 2000;
const USER_MESSAGE_RESERVE_TOKENS = 500;
const DEFAULT_MAX_CREDITS = "1000";
const DEFAULT_MAX_TOKENS = 4000;
const DEFAULT_TIMEOUT_MS = 60000;

// ============= Redis Cache Types =============

/**
 * Represents a single message entry in the per-chat history map.
 */
interface HistoryEntry {
    /** The unique identifier of the message. */
    id: string;
    /** The identifier of the parent message, if any. */
    parentId: string | null;
    /** The identifier of the user who sent this message. */
    userId: string | null;
    /** The raw text content of the message. */
    text: string;
    /** The message configuration (metadata) stored in the database. */
    config: unknown;
    /** The language code of the message. */
    language: string;
    /** ISO timestamp of when the message was created. */
    createdAt: string;
    /** Precomputed token size for this message. */
    tokenSize: number;
}

/**
 * A map of messageId to HistoryEntry for a chat.
 */
type HistoryMap = Record<string, HistoryEntry>;

// ============= Token Budget Types =============

interface TokenBudget {
    totalAvailable: number;
    systemMessageBudget: number;
    historyBudget: number;
    responseReserve: number;
    userMessageReserve: number;
    safetyBuffer: number;
}

// ============= Main Implementation =============

export class MessageHistoryBuilder {
    private static instance: MessageHistoryBuilder | null = null;
    private readonly cache: ChatContextCache;
    private readonly toolRegistry: ToolRegistry;

    private constructor() {
        this.cache = ChatContextCache.get();
        this.toolRegistry = new ToolRegistry();
    }

    /**
     * Get the singleton instance of MessageHistoryBuilder
     */
    static get(): MessageHistoryBuilder {
        if (!MessageHistoryBuilder.instance) {
            MessageHistoryBuilder.instance = new MessageHistoryBuilder();
        }
        return MessageHistoryBuilder.instance;
    }

    /**
     * Build complete message array ready for LLM
     * 
     * Returns: [systemMessage, ...conversationHistory]
     * This is the ONLY method ResponseService needs to call.
     */
    async buildMessages(context: ResponseContext): Promise<ChatMessage[]> {
        const startTime = Date.now();

        try {
            // 1. Calculate token budgets
            const budget = this.calculateTokenBudget(context);

            // 2. Build system message (always first)
            const systemMessage = await this.buildSystemMessage(context, budget.systemMessageBudget);

            // 3. Load conversation history within remaining budget
            const historyMessages = await this.loadConversationHistory(
                context.swarmId,
                budget.historyBudget,
                context.bot.config.model || "gpt-4",
            );

            logger.debug("MessageHistoryBuilder: Messages built successfully", {
                duration: Date.now() - startTime,
                systemTokens: budget.systemMessageBudget,
                historyMessages: historyMessages.length,
                totalBudget: budget.totalAvailable,
            });

            // 4. Return complete message array for LLM
            return [systemMessage, ...historyMessages];

        } catch (error) {
            logger.error("MessageHistoryBuilder: Failed to build messages", {
                error: error instanceof Error ? error.message : String(error),
                duration: Date.now() - startTime,
            });
            throw error;
        }
    }

    /**
     * Calculate token budget allocation
     */
    private calculateTokenBudget(context: ResponseContext): TokenBudget {
        // Get model limit from AI service registry
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(context.bot.config.model || "gpt-4");
        const service = registry.getService(serviceId);
        const modelLimit = service.getContextSize(context.bot.config.model || "gpt-4");

        const responseReserve = RESPONSE_RESERVE_TOKENS; // Reserve for LLM response
        const userMessageReserve = USER_MESSAGE_RESERVE_TOKENS; // Reserve for next user message
        const safetyBuffer = Math.floor(modelLimit * CONTEXT_SIZE_SAFETY_BUFFER_PERCENT);

        const totalAvailable = modelLimit - responseReserve - userMessageReserve - safetyBuffer;

        // System message gets 25% of available tokens
        const systemMessageBudget = Math.floor(totalAvailable * SYSTEM_MESSAGE_BUDGET_PERCENT);
        const historyBudget = totalAvailable - systemMessageBudget;

        return {
            totalAvailable,
            systemMessageBudget,
            historyBudget,
            responseReserve,
            userMessageReserve,
            safetyBuffer,
        };
    }

    /**
     * Build system message using PromptService
     */
    private async buildSystemMessage(
        context: ResponseContext,
        tokenBudget: number,
    ): Promise<ChatMessage> {
        // Convert ResponseContext to PromptContext for PromptService
        const promptContext = this.convertToPromptContext(context);

        // Check for agent-specific direct prompt content
        const agentSpec = context.bot.config.agentSpec;
        let systemContent: string;

        if (agentSpec?.prompt?.source === "direct" && agentSpec.prompt.content) {
            logger.debug("MessageHistoryBuilder: Using agent-specific direct prompt", {
                botId: context.bot.id,
                promptMode: agentSpec.prompt.mode,
            });

            // Use PromptService with direct prompt content
            systemContent = await PromptService.buildSystemMessage(promptContext, {
                directPromptContent: agentSpec.prompt.content,
                userData: context.userData,
            });
        } else {
            // Use PromptService for sophisticated prompt generation
            systemContent = await PromptService.buildSystemMessage(promptContext, {
                userData: context.userData,
            });
        }

        // Check if it fits in budget and truncate if needed
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(context.bot.config.model || "gpt-4");
        const service = registry.getService(serviceId);
        const systemTokens = service.estimateTokens({
            aiModel: context.bot.config.model || "gpt-4",
            text: systemContent,
        }).tokens;

        if (systemTokens > tokenBudget) {
            logger.warn("MessageHistoryBuilder: System message exceeds budget, truncating", {
                systemTokens,
                tokenBudget,
            });
            systemContent = await this.truncateSystemMessage(systemContent, tokenBudget, context.bot.config.model || "gpt-4");
        }

        // Return as ChatMessage
        return {
            __typename: "ChatMessage" as const,
            id: "system",
            createdAt: new Date().toISOString(),
            config: {
                __version: "1.0",
                role: "system",
            } as MessageConfigObject,
            text: systemContent,
            user: { id: "system" },
            language: "en",
        } as ChatMessage;
    }

    /**
     * Convert ResponseContext to PromptContext for PromptService
     */
    private convertToPromptContext(context: ResponseContext): PromptContext {
        // Extract goal from context or use default
        const effectiveGoal = context.bot.config.agentSpec?.goal || "Process current event.";

        // Create conversation config from context data
        const convoConfig = {
            __version: "1.0" as const,
            teamId: context.swarmId,
            swarmLeader: context.bot.id,
            subtasks: [],
            subtaskLeaders: {},
            eventSubscriptions: {},
            blackboard: [],
            resources: [],
            records: [],
            stats: {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: null,
                lastProcessingCycleEndedAt: null,
            },
            limits: {
                maxCredits: context.constraints?.maxCredits || DEFAULT_MAX_CREDITS,
                maxTokens: context.constraints?.maxTokens || DEFAULT_MAX_TOKENS,
                timeoutMs: context.constraints?.timeoutMs || DEFAULT_TIMEOUT_MS,
            },
            pendingToolCalls: [],
            secrets: {},
        };

        return {
            goal: effectiveGoal,
            bot: context.bot,
            convoConfig,
            teamConfig: undefined,
            toolRegistry: this.toolRegistry,
            userId: context.userData?.id,
            swarmId: context.swarmId,
            input: {
                conversationHistory: context.conversationHistory,
                availableTools: context.availableTools,
                strategy: context.strategy,
                constraints: context.constraints,
            },
            swarmState: context.swarmState,
        };
    }

    /**
     * Load conversation history within token budget
     * Uses Redis caching and database fallback
     */
    private async loadConversationHistory(
        chatId: string,
        tokenBudget: number,
        aiModel: string,
    ): Promise<ChatMessage[]> {
        // 1. Try Redis cache first
        let map: HistoryMap | null = await this.cache.getHistoryMap(chatId);

        if (!map) {
            // 2. Load from database and cache
            map = await this.loadFromDatabaseAndCache(chatId, aiModel);
        }

        // 3. Ensure token sizes are present
        await this.ensureTokenSizes(map, aiModel, chatId);

        // 4. Select messages within budget using parent-child traversal
        const selectedEntries = this.selectMessagesWithinBudget(map, tokenBudget);

        // 5. Convert to ChatMessage format
        return this.convertHistoryEntriesToChatMessages(selectedEntries);
    }

    /**
     * Load messages from database and cache them
     */
    private async loadFromDatabaseAndCache(chatId: string, aiModel: string): Promise<HistoryMap> {
        // Fetch all messages from the DB
        let recs = await DbProvider.get().chat_message.findMany({
            where: { chatId: BigInt(chatId) },
            include: { parent: { select: { id: true } }, user: { select: { id: true } } },
            orderBy: { createdAt: "asc" },
        });

        // Trim to the most recent entries if over limit
        if (recs.length > MAX_HISTORY_ENTRIES) {
            recs = recs.slice(-MAX_HISTORY_ENTRIES);
        }

        // Build HistoryMap
        const map: HistoryMap = {};
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(aiModel);
        const service = registry.getService(serviceId);
        const tokenCounter = new TokenCounter(service, aiModel);

        for (const rec of recs) {
            const msg: MessageState = {
                id: rec.id.toString(),
                createdAt: rec.createdAt.toISOString(),
                text: rec.text ?? "",
                language: rec.language,
                config: rec.config as any,
                parent: rec.parent?.id ? { id: rec.parent.id.toString() } : null,
                user: rec.user?.id ? { id: rec.user.id.toString() } : null,
            };

            const { size } = tokenCounter.ensure(msg, {});

            map[msg.id] = {
                id: msg.id,
                parentId: msg.parent?.id ?? null,
                userId: msg.user?.id ?? null,
                text: msg.text,
                config: msg.config,
                language: msg.language,
                createdAt: msg.createdAt,
                tokenSize: size,
            };
        }

        // Cache the result
        await this.cache.setHistoryMap(chatId, map);
        return map;
    }

    /**
     * Ensure all entries have token sizes computed
     */
    private async ensureTokenSizes(map: HistoryMap, aiModel: string, chatId: string): Promise<void> {
        let dirty = false;
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(aiModel);
        const service = registry.getService(serviceId);
        const tokenCounter = new TokenCounter(service, aiModel);

        for (const entry of Object.values(map)) {
            if (entry.tokenSize <= 0) {
                const dummy: MessageState = {
                    id: entry.id,
                    createdAt: entry.createdAt,
                    text: entry.text,
                    language: entry.language,
                    config: entry.config as any,
                    parent: entry.parentId ? { id: entry.parentId } : null,
                    user: entry.userId ? { id: entry.userId } : null,
                };

                const { size } = tokenCounter.ensure(dummy, {});
                entry.tokenSize = size;
                dirty = true;
            }
        }

        if (dirty) {
            await this.cache.setHistoryMap(chatId, map);
        }
    }

    /**
     * Select messages within budget using parent-child traversal
     */
    private selectMessagesWithinBudget(map: HistoryMap, tokenBudget: number): HistoryEntry[] {
        // Walk backwards via parentId under token budget
        const entries = Object.values(map).sort(
            (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );

        const collected: HistoryEntry[] = [];
        let tokens = 0;
        let curId = entries.length ? entries[entries.length - 1].id : null;

        while (curId) {
            const e = map[curId];
            if (!e || tokens + e.tokenSize > tokenBudget) break;

            collected.push(e);
            tokens += e.tokenSize;
            curId = e.parentId;
        }

        collected.reverse();
        return collected;
    }

    /**
     * Convert HistoryEntry[] to ChatMessage[]
     */
    private async convertHistoryEntriesToChatMessages(
        entries: HistoryEntry[],
    ): Promise<ChatMessage[]> {
        if (entries.length === 0) {
            return [];
        }

        // Fetch full messages from database
        const collectedIds = entries.map(e => BigInt(e.id));
        const recs = await DbProvider.get().chat_message.findMany({
            where: { id: { in: collectedIds } },
            include: { parent: { select: { id: true } }, user: { select: { id: true } } },
            orderBy: { createdAt: "asc" },
        });

        // Convert to ChatMessage format
        return recs.map(rec => ({
            __typename: "ChatMessage" as const,
            id: rec.id.toString(),
            createdAt: rec.createdAt.toISOString(),
            config: rec.config as unknown as MessageConfigObject,
            text: rec.text ?? "",
            user: rec.user?.id ? { id: rec.user.id.toString() } : { id: "unknown" },
            language: rec.language,
        } as ChatMessage));
    }

    // ============= Helper Methods =============

    /**
     * Truncate system message if it exceeds budget
     */
    private async truncateSystemMessage(content: string, maxTokens: number, aiModel: string): Promise<string> {
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(aiModel);
        const service = registry.getService(serviceId);

        // Binary search for the right length
        let low = 0;
        let high = content.length;

        while (low < high) {
            const mid = Math.floor((low + high + 1) / 2);
            const candidate = content.substring(0, mid) + "\n\n[System message truncated due to token limits]";
            const tokens = service.estimateTokens({ aiModel, text: candidate }).tokens;

            if (tokens <= maxTokens) {
                low = mid;
            } else {
                high = mid - 1;
            }
        }

        return content.substring(0, low) + "\n\n[System message truncated due to token limits]";
    }
}


