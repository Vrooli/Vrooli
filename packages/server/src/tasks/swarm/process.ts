import { generatePK, toBotId, toSwarmId, type BotParticipant, type ChatMessage, type ConversationContext, type ConversationParams, type MessageConfigObject, type MessageState } from "@vrooli/shared";
import { type Job } from "bullmq";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ConversationEngine } from "../../services/conversation/conversationEngine.js";
import { SwarmContextManager } from "../../services/execution/shared/SwarmContextManager.js";
import { SwarmStateMachine } from "../../services/execution/tier1/swarmStateMachine.js";
import { CachedConversationStateStore, PrismaChatStore } from "../../services/response/chatStore.js";
import { RedisMessageStore } from "../../services/response/messageStore.js";
import { ResponseService } from "../../services/response/responseService.js";
import { getDefaultLlmRouter, getDefaultToolRunner } from "../../services/response/services.js";
import { CacheService } from "../../redisConn.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask, type SwarmExecutionTask } from "../taskTypes.js";

// Default configuration constants
const DEFAULT_MAX_TOKENS = 4000;
const DEFAULT_TEMPERATURE = 0.7;
const MAX_CONVERSATION_HISTORY = 20;


export type ActiveSwarmRecord = BaseActiveTaskRecord;
class SwarmRegistry extends BaseActiveTaskRegistry<ActiveSwarmRecord, SwarmStateMachine> {
    // Add any swarm-specific methods here
}
export const activeSwarmRegistry = new SwarmRegistry();


export async function llmProcessBotMessage(payload: LLMCompletionTask): Promise<{ success: boolean; messageId?: string }> {
    logger.info("[llmProcessBotMessage] Processing direct bot response via ConversationEngine", {
        chatId: payload.chatId,
        messageId: payload.messageId,
        userId: payload.userData.id,
    });

    try {
        // 1. Build conversation context
        const conversationContext = await buildConversationContext(payload);

        // 2. Create conversation params
        const conversationParams: ConversationParams = {
            context: conversationContext,
            trigger: {
                type: "user_message",
                message: await loadTriggerMessage(payload.messageId),
            },
            strategy: "conversation",
            constraints: {
                maxTurns: 1, // Single response turn
                timeoutMs: 60000, // 60 second timeout
                maxParticipants: 1, // Only one bot should respond
                allowToolUse: true, // Allow tool use if needed
            },
        };

        // 3. Create ConversationEngine with streaming-enabled ResponseService
        const [toolRunner, contextBuilder] = await Promise.all([
            getDefaultToolRunner(),
            getDefaultContextBuilder(),
        ]);

        const responseService = new ResponseService(
            getDefaultLlmRouter(),
            toolRunner,
            contextBuilder,
            {
                enableStreaming: true,
                streamingChatId: payload.chatId, // Enable streaming for this chat
            },
        );

        const contextManager = new SwarmContextManager();
        const conversationEngine = new ConversationEngine(
            responseService,
            contextManager,
        );

        // 4. Generate response (streaming happens automatically via ResponseService)
        const result = await conversationEngine.orchestrateConversation(conversationParams);

        if (!result.success || result.messages.length === 0) {
            throw new Error("Failed to generate bot response");
        }

        // 5. Persist to database
        const botMessage = result.messages[0];
        const dbMessage = await DbProvider.get().chat_message.create({
            data: {
                chatId: BigInt(payload.chatId),
                parentId: BigInt(payload.messageId),
                createdAt: new Date(),
                updatedAt: new Date(),
                user: {
                    connect: { id: BigInt(botMessage.user.id) },
                },
                parent: {
                    connect: { id: BigInt(payload.messageId) },
                },
                chat: {
                    connect: { id: BigInt(payload.chatId) },
                },
                config: {
                    id: generatePK().toString(),
                    role: "bot",
                    text: botMessage.text,
                    __version: "1.0.0",
                },
                translations: [{
                    id: generatePK().toString(),
                    language: payload.userData.languages?.[0] || "en",
                    text: botMessage.text,
                }],
            },
        });

        // 6. Update cache (after DB persistence as per interface contract)
        await RedisMessageStore.get().addMessage(payload.chatId, {
            id: dbMessage.id.toString(),
            chatId: payload.chatId,
            createdAt: dbMessage.createdAt,
            updatedAt: dbMessage.updatedAt,
            config: dbMessage.config as unknown as MessageConfigObject,
            parent: payload.messageId ? { id: payload.messageId } : null,
            user: { id: botMessage.user.id },
            translations: dbMessage.translations.map(t => ({
                id: t.id,
                language: t.language,
                text: t.text,
            })),
        } as MessageState);

        // 7. Socket events are already handled by Trigger system in chatMessage.create

        logger.info("[llmProcessBotMessage] Successfully processed bot response", {
            chatId: payload.chatId,
            messageId: dbMessage.id.toString(),
        });

        return { success: true, messageId: dbMessage.id.toString() };

    } catch (error) {
        logger.error("[llmProcessBotMessage] Failed to process bot response", {
            chatId: payload.chatId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

/**
 * Process swarm execution directly through SwarmStateMachine
 * 
 * This uses SwarmStateMachine directly with SwarmExecutionTask structure
 * for clean integration with the three-tier architecture.
 */
async function processSwarmExecution(payload: SwarmExecutionTask) {
    logger.info("[processSwarmExecution] Starting swarm execution", {
        swarmId: payload.context.swarmId,
        userId: payload.context.userData.id,
    });

    try {
        // Extract swarm ID or generate new one if not provided
        const swarmId = payload.context.swarmId || generatePK().toString();

        const responseService = new ResponseService();
        const contextManager = new SwarmContextManager();
        const conversationEngine = new ConversationEngine(responseService, contextManager);
        
        // Create chat store for loading chat configuration
        const messageStore = new RedisMessageStore(CacheService.get());
        const chatStore = new CachedConversationStateStore(
            new PrismaChatStore(), 
            CacheService.get(),
            messageStore,
        );
        
        const coordinator = new SwarmStateMachine(
            contextManager,
            conversationEngine,
            responseService,
            chatStore,
        );

        // Start swarm with SwarmExecutionTask directly
        const result = await coordinator.start(payload);
        if (!result.success) {
            throw new Error(result.error?.message || "Failed to start swarm");
        }

        // Create registry record
        const record: ActiveSwarmRecord = {
            id: swarmId,
            userId: payload.context.userData.id,
            hasPremium: payload.context.userData.hasPremium || false,
            startTime: Date.now(),
        };

        // Add coordinator to registry
        activeSwarmRegistry.add(record, coordinator);

        logger.info("[processSwarmExecution] Successfully started swarm", {
            swarmId,
        });

        return { swarmId };

    } catch (error) {
        logger.error("[processSwarmExecution] Failed to start swarm execution", {
            swarmId: payload.context.swarmId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

export async function llmProcess({ data }: Job<LLMCompletionTask | SwarmExecutionTask>) {
    switch (data.type) {
        case QueueTaskType.LLM_COMPLETION:
            return llmProcessBotMessage(data as LLMCompletionTask);

        case QueueTaskType.SWARM_EXECUTION:
            return processSwarmExecution(data as SwarmExecutionTask);

        default:
            throw new CustomError("0330", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}

/**
 * Helper function to build conversation context from LLM completion task
 */
async function buildConversationContext(payload: LLMCompletionTask): Promise<ConversationContext> {
    // Load chat and participants from database
    const chat = await DbProvider.get().chat.findUnique({
        where: { id: BigInt(payload.chatId) },
        include: {
            participants: {
                include: {
                    user: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                            isBot: true,
                            botSettings: true,
                        },
                    },
                },
            },
            config: true,
        },
    });

    if (!chat) {
        throw new Error(`Chat ${payload.chatId} not found`);
    }

    // Build bot participants list
    const botParticipants: BotParticipant[] = [];

    // If a specific bot is requested, find it
    if (payload.respondingBot?.id) {
        const botParticipant = chat.participants.find(p =>
            p.user.id.toString() === payload.respondingBot?.id && p.user.isBot,
        );
        if (botParticipant) {
            botParticipants.push({
                id: toBotId(botParticipant.user.id.toString()),
                name: botParticipant.user.name || "Bot",
                config: {
                    id: botParticipant.user.id.toString(),
                    name: botParticipant.user.name || "Bot",
                    model: payload.model || botParticipant.user.botSettings?.model || "gpt-4",
                    description: botParticipant.user.botSettings?.description || "AI Assistant",
                    maxTokens: botParticipant.user.botSettings?.maxTokens || DEFAULT_MAX_TOKENS,
                    temperature: botParticipant.user.botSettings?.temperature || DEFAULT_TEMPERATURE,
                    specializations: botParticipant.user.botSettings?.specializations || [],
                },
                state: {
                    isProcessing: false,
                    isWaiting: true,
                    hasResponded: false,
                },
                isAvailable: true,
            });
        }
    } else {
        // Find all bot participants in the chat
        for (const participant of chat.participants) {
            if (participant.user.isBot) {
                botParticipants.push({
                    id: toBotId(participant.user.id.toString()),
                    name: participant.user.name || "Bot",
                    config: {
                        id: participant.user.id.toString(),
                        name: participant.user.name || "Bot",
                        model: payload.model || participant.user.botSettings?.model || "gpt-4",
                        description: participant.user.botSettings?.description || "AI Assistant",
                        maxTokens: participant.user.botSettings?.maxTokens || DEFAULT_MAX_TOKENS,
                        temperature: participant.user.botSettings?.temperature || DEFAULT_TEMPERATURE,
                        specializations: participant.user.botSettings?.specializations || [],
                    },
                    state: {
                        isProcessing: false,
                        isWaiting: true,
                        hasResponded: false,
                    },
                    isAvailable: true,
                });
            }
        }
    }

    // Load conversation history (last N messages for context)
    const messages = await DbProvider.get().chat_message.findMany({
        where: { chatId: BigInt(payload.chatId) },
        orderBy: { createdAt: "desc" },
        take: MAX_CONVERSATION_HISTORY,
        include: {
            user: {
                select: {
                    id: true,
                    name: true,
                    handle: true,
                },
            },
            translations: true,
        },
    });

    // Convert to ChatMessage format and reverse to chronological order
    const conversationHistory: ChatMessage[] = messages.reverse().map(msg => ({
        id: msg.id.toString(),
        createdAt: msg.createdAt,
        config: msg.config as MessageConfigObject,
        text: msg.translations?.[0]?.text || "",
        user: {
            id: msg.user.id.toString(),
            name: msg.user.name || undefined,
            handle: msg.user.handle || undefined,
        },
    }));

    // Load available tools (for now, return empty array - can be enhanced later)
    const availableTools = [];

    return {
        swarmId: toSwarmId(payload.chatId), // Use chatId as swarmId for simple conversations
        userData: payload.userData,
        timestamp: new Date(),
        participants: botParticipants,
        conversationHistory,
        availableTools,
        teamConfig: undefined, // No team config for simple conversations
        sharedState: {}, // No shared state needed for simple conversations
    };
}

/**
 * Helper function to load the trigger message
 */
async function loadTriggerMessage(messageId: string): Promise<ChatMessage> {
    const message = await DbProvider.get().chat_message.findUnique({
        where: { id: BigInt(messageId) },
        include: {
            user: {
                select: {
                    id: true,
                    name: true,
                    handle: true,
                },
            },
            translations: true,
        },
    });

    if (!message) {
        throw new Error(`Message ${messageId} not found`);
    }

    return {
        id: message.id.toString(),
        createdAt: message.createdAt.toISOString(),
        config: message.config as MessageConfigObject,
        text: message.translations?.[0]?.text || "",
        user: {
            id: message.user.id.toString(),
            name: message.user.name || undefined,
            handle: message.user.handle || undefined,
        },
    };
}
