/**
 * Unified Prompt Builder - Clean LLM Message Array Generation
 * 
 * This service provides a single method that ResponseService needs:
 * buildMessages() returns a complete message array ready for any LLM.
 * 
 * Message Array Format:
 * [0] = System message (built from ResponseService + ResponseContext)
 * [1..n] = Conversation history (token-budgeted, Redis-cached)
 */

import type { ChatMessage, MessageConfigObject, MessageState, ResponseContext } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { ToolRegistry } from "../mcp/registry.js";
import { ChatContextCache, TokenCounter } from "./messageStore.js";
import { AIServiceRegistry } from "./registry.js";
import { ResponseService, type PromptContext } from "./responseService.js";
import { MessageTypeAdapters } from "./typeAdapters.js";

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
    async buildMessages(context: ResponseContext): Promise<MessageState[]> {
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
                context.bot.config.modelConfig?.preferredModel || "gpt-4",
            );

            // Message history built

            // 4. Return complete message array as MessageState[]
            const allMessages = [systemMessage, ...historyMessages];
            return allMessages;

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
        const serviceId = registry.getServiceId(context.bot.config.modelConfig?.preferredModel || "gpt-4");
        const service = registry.getService(serviceId);
        const modelLimit = service.getContextSize(context.bot.config.modelConfig?.preferredModel || "gpt-4");

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
     * Build system message using ResponseService
     */
    private async buildSystemMessage(
        context: ResponseContext,
        tokenBudget: number,
    ): Promise<MessageState> {
        // Convert ResponseContext to PromptContext for ResponseService
        const promptContext = this.convertToPromptContext(context);

        // Check for agent-specific direct prompt content
        const agentSpec = context.bot.config.agentSpec;
        let systemContent: string;

        if (agentSpec?.prompt?.source === "direct" && agentSpec.prompt.content) {
            logger.debug("MessageHistoryBuilder: Using agent-specific direct prompt", {
                botId: context.bot.id,
                promptMode: agentSpec.prompt.mode,
            });

            // Use ResponseService with direct prompt content
            systemContent = await ResponseService.buildSystemMessage(promptContext, {
                directPromptContent: agentSpec.prompt.content,
                userData: context.userData,
            });
        } else {
            // Use ResponseService for sophisticated prompt generation
            systemContent = await ResponseService.buildSystemMessage(promptContext, {
                userData: context.userData,
            });
        }

        // Check if it fits in budget and truncate if needed
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(context.bot.config.modelConfig?.preferredModel || "gpt-4");
        const service = registry.getService(serviceId);
        const systemTokens = service.estimateTokens({
            aiModel: context.bot.config.modelConfig?.preferredModel || "gpt-4",
            text: systemContent,
        }).tokens;

        if (systemTokens > tokenBudget) {
            logger.warn("MessageHistoryBuilder: System message exceeds budget, truncating", {
                systemTokens,
                tokenBudget,
            });
            systemContent = await this.truncateSystemMessage(systemContent, tokenBudget, context.bot.config.modelConfig?.preferredModel || "gpt-4");
        }

        // Return as MessageState using type adapters
        return MessageTypeAdapters.createMessageState({
            id: "system",
            text: systemContent,
            role: "system",
            userId: "system",
            language: "en",
        });
    }

    /**
     * Convert ResponseContext to PromptContext for ResponseService
     */
    private convertToPromptContext(context: ResponseContext): PromptContext {
        // Extract goal from context or use default
        const effectiveGoal = context.bot.config.agentSpec?.goal || "Process current event.";

        // Use the chatConfig from swarmState if available, otherwise create a minimal one
        const convoConfig = context.swarmState?.chatConfig || {
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

        // TODO: If teamId is present in convoConfig, we should load the team data
        // For now, we'll leave team as undefined to match the current behavior
        // In the future, this should be enhanced to load actual team data from the database
        return {
            goal: effectiveGoal,
            bot: context.bot,
            convoConfig,
            team: undefined,
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
    ): Promise<MessageState[]> {
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

        // 5. Convert to MessageState format
        return this.convertHistoryEntriesToMessageStates(selectedEntries);
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
    private async convertHistoryEntriesToMessageStates(
        entries: HistoryEntry[],
    ): Promise<MessageState[]> {
        if (entries.length === 0) {
            return [];
        }

        // Fetch full messages from database as proper ChatMessage objects
        const collectedIds = entries.map(e => BigInt(e.id));
        const chatMessages = await this.loadFullChatMessages(collectedIds);

        // Convert ChatMessage[] to MessageState[] using safe type adapters
        return chatMessages.map(msg => MessageTypeAdapters.chatMessageToMessageState(msg));
    }

    /**
     * Load full ChatMessage objects from database with all required relations
     * 
     * This method properly constructs ChatMessage objects with all the fields
     * required by the interface, ensuring type safety and data integrity.
     */
    private async loadFullChatMessages(messageIds: bigint[]): Promise<ChatMessage[]> {
        const recs = await DbProvider.get().chat_message.findMany({
            where: { id: { in: messageIds } },
            orderBy: { createdAt: "asc" },
        });

        // Convert database records to proper ChatMessage objects
        return recs.map(rec => ({
            __typename: "ChatMessage" as const,
            id: rec.id.toString(),
            createdAt: rec.createdAt.toISOString(),
            updatedAt: rec.updatedAt.toISOString(),
            config: rec.config as unknown as MessageConfigObject,
            text: rec.text ?? "",
            language: rec.language,
            versionIndex: rec.versionIndex,
            sequence: 0, // Default value since sequence field doesn't exist in database
            score: rec.score,
            reportsCount: 0, // Default value
            parent: rec.parentId ? {
                __typename: "ChatMessageParent" as const,
                id: rec.parentId.toString(),
                createdAt: rec.createdAt.toISOString(),
            } : undefined,
            user: {
                __typename: "User" as const,
                id: rec.userId?.toString() || "unknown",
                name: "",
                handle: "",
            } as any, // Simplified User object
            chat: {
                __typename: "Chat" as const,
                id: rec.chatId.toString(),
                name: "",
            } as any, // Simplified Chat object
            reactionSummaries: [],
            reports: [],
            you: {
                __typename: "ChatMessageYou" as const,
                canDelete: false,
                canUpdate: false,
                canReply: true,
                canReport: false,
                canReact: true,
                reaction: null,
            },
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


