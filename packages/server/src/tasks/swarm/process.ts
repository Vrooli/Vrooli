import { generatePK, toBotId, toSwarmId, type BotParticipant, type ChatMessage, type ConversationContext, type ConversationParams, type MessageConfigObject, type MessageState } from "@vrooli/shared";
import { type Job } from "bullmq";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ConversationEngine } from "../../services/conversation/conversationEngine.js";
import { SwarmContextManager } from "../../services/execution/shared/SwarmContextManager.js";
import { SwarmStateMachine } from "../../services/execution/tier1/swarmStateMachine.js";
import { ResponseService } from "../../services/response/responseService.js";
import { getDefaultLlmRouter, getDefaultToolRunner } from "../../services/response/services.js";
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
            config: dbMessage.config as MessageConfigObject,
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

        // Use the existing service instances from responseEngine
        const coordinator = new SwarmStateMachine(
            swarmContextManager,
            conversationEngine,
            responseService,
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
