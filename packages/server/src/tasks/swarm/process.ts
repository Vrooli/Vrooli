import { type Prisma } from "@prisma/client";
import { generatePK, LATEST_CONFIG_VERSION, MessageConfig, toBotId, toSwarmId, type BotConfigObject, type BotParticipant, type ChatMessage, type ConversationContext, type ConversationParams, type MessageConfigObject, type MessageState } from "@vrooli/shared";
import { type Job } from "bullmq";
import { getUserLanguages } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ConversationEngine } from "../../services/conversation/conversationEngine.js";
import { SwarmContextManager } from "../../services/execution/shared/SwarmContextManager.js";
import { SwarmStateMachine } from "../../services/execution/tier1/swarmStateMachine.js";
import { CachedConversationStateStore, PrismaChatStore } from "../../services/response/chatStore.js";
import { RedisMessageStore } from "../../services/response/messageStore.js";
import { ResponseService } from "../../services/response/responseService.js";
import { MessageTypeAdapters } from "../../services/response/typeAdapters.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask, type SwarmExecutionTask } from "../taskTypes.js";

// Default configuration constants
const MAX_CONVERSATION_HISTORY = 20;

// Type guards for task validation
function isValidLLMCompletionTask(payload: unknown): payload is LLMCompletionTask {
    return typeof payload === "object" &&
        payload !== null &&
        "chatId" in payload &&
        "messageId" in payload &&
        "userData" in payload &&
        "allocation" in payload &&
        "type" in payload &&
        payload.type === QueueTaskType.LLM_COMPLETION;
}

function isValidSwarmExecutionTask(payload: unknown): payload is SwarmExecutionTask {
    return typeof payload === "object" &&
        payload !== null &&
        "context" in payload &&
        "input" in payload &&
        "type" in payload &&
        payload.type === QueueTaskType.SWARM_EXECUTION;
}

export type ActiveSwarmRecord = BaseActiveTaskRecord;
class SwarmRegistry extends BaseActiveTaskRegistry<ActiveSwarmRecord, SwarmStateMachine> {
    // Add any swarm-specific methods here
}
export const activeSwarmRegistry = new SwarmRegistry();


export async function llmProcessBotMessage(payload: LLMCompletionTask): Promise<{ success: boolean; messageId?: string; swarmId?: string }> {
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
            strategy: "conversational",
            constraints: {
                maxTurns: 1, // Single response turn
                timeoutMs: 60000, // 60 second timeout
                maxParticipants: 1, // Only one bot should respond
                allowToolUse: true, // Allow tool use if needed
            },
        };

        const responseService = new ResponseService({
            enableStreaming: true,
            streamingChatId: payload.chatId, // Enable streaming for this chat
        });

        const contextManager = new SwarmContextManager(payload);
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
                id: generatePK(),
                chat: { connect: { id: BigInt(payload.chatId) } },
                parent: { connect: { id: BigInt(payload.messageId) } },
                user: { connect: { id: BigInt(botMessage.user?.id || "0") } },
                config: (new MessageConfig({ config: { __version: LATEST_CONFIG_VERSION, role: "assistant" } })).export() as unknown as Prisma.InputJsonValue,
                text: botMessage.text,
                language: getUserLanguages(payload.userData.languages)[0],
            },
            select: {
                id: true,
                createdAt: true,
                updatedAt: true,
                config: true,
                text: true,
                language: true,
                user: {
                    select: {
                        id: true,
                    },
                },
                parent: {
                    select: {
                        id: true,
                    },
                },
                chat: {
                    select: {
                        id: true,
                    },
                },
            },
        });

        // 6. Update cache (after DB persistence as per interface contract)
        const messageState: MessageState = {
            id: dbMessage.id.toString(),
            createdAt: dbMessage.createdAt.toISOString(),
            config: dbMessage.config as unknown as MessageConfigObject,
            text: dbMessage.text,
            language: dbMessage.language,
            parent: payload.messageId ? { id: payload.messageId } : null,
            user: { id: botMessage.user?.id?.toString() || "0" },
        };
        await RedisMessageStore.get().addMessage(payload.chatId, messageState);

        return { success: true, messageId: dbMessage.id.toString(), swarmId: payload.chatId };

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
    // Starting swarm execution

    try {
        // Extract swarm ID or generate new one if not provided
        const swarmId = payload.input.swarmId || generatePK().toString();

        const responseService = new ResponseService();
        const contextManager = new SwarmContextManager(payload);
        const conversationEngine = new ConversationEngine(responseService, contextManager);

        // Create chat store for loading chat configuration
        const chatStore = new CachedConversationStateStore(new PrismaChatStore());

        const coordinator = new SwarmStateMachine(
            contextManager,
            conversationEngine,
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
            userId: payload.input.userData.id,
            hasPremium: payload.input.userData.hasPremium || false,
            startTime: Date.now(),
        };

        // Add coordinator to registry
        activeSwarmRegistry.add(record, coordinator);

        // Swarm started successfully

        return { success: true, swarmId, messageId: undefined };

    } catch (error) {
        logger.error("[processSwarmExecution] Failed to start swarm execution", {
            swarmId: payload.input.swarmId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

export async function llmProcess({ data }: Job<LLMCompletionTask | SwarmExecutionTask>) {
    switch (data.type) {
        case QueueTaskType.LLM_COMPLETION:
            if (!isValidLLMCompletionTask(data)) {
                throw new CustomError("0330", "InvalidArgs", {
                    reason: "Invalid LLM completion task structure",
                    required: ["chatId", "messageId", "userData"],
                });
            }
            return llmProcessBotMessage(data);

        case QueueTaskType.SWARM_EXECUTION:
            if (!isValidSwarmExecutionTask(data)) {
                throw new CustomError("0330", "InvalidArgs", {
                    reason: "Invalid swarm execution task structure",
                    required: ["userData", "input.goal"],
                });
            }
            return processSwarmExecution(data);

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
        select: {
            id: true,
            config: true,
            participants: {
                select: {
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
                config: botParticipant.user.botSettings as unknown as BotConfigObject,
                state: "ready",
            });
        }
    } else {
        // Find all bot participants in the chat
        for (const participant of chat.participants) {
            if (participant.user.isBot) {
                botParticipants.push({
                    id: toBotId(participant.user.id.toString()),
                    name: participant.user.name || "Bot",
                    config: participant.user.botSettings as unknown as BotConfigObject,
                    state: "ready",
                });
            }
        }
    }

    // Load conversation history (last N messages for context)
    const messages = await DbProvider.get().chat_message.findMany({
        where: { chatId: BigInt(payload.chatId) },
        orderBy: { createdAt: "desc" },
        take: MAX_CONVERSATION_HISTORY,
        select: {
            id: true,
            config: true,
            createdAt: true,
            language: true,
            text: true,
            user: {
                select: {
                    id: true,
                    name: true,
                    handle: true,
                },
            },
        },
    });

    // Load full ChatMessage objects and convert to MessageState[]
    const fullChatMessages = await loadFullChatMessages(messages.map(m => BigInt(m.id)));
    const conversationHistory: MessageState[] = fullChatMessages.map(msg =>
        MessageTypeAdapters.chatMessageToMessageState(msg),
    );

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
 * Load full ChatMessage objects from database with all required relations
 * TODO probably shouldn't have this
 */
async function loadFullChatMessages(messageIds: bigint[]): Promise<ChatMessage[]> {
    if (messageIds.length === 0) {
        return [];
    }

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
        sequence: 0, // Default value
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

/**
 * Helper function to load the trigger message
 */
async function loadTriggerMessage(messageId: string): Promise<ChatMessage> {
    const fullChatMessages = await loadFullChatMessages([BigInt(messageId)]);

    if (fullChatMessages.length === 0) {
        throw new Error(`Message ${messageId} not found`);
    }

    return fullChatMessages[0];
}
