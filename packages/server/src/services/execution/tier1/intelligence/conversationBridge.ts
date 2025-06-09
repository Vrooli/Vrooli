import { type Logger } from "winston";
import { generatePK, type ChatMessage, type BotParticipant } from "@vrooli/shared";
import { completionService } from "../../../conversation/responseEngine.js";
import { type LLMCompletionTask } from "../../../../tasks/taskTypes.js";
import { PromptTemplateService } from "./promptService.js";

/**
 * Conversation Bridge
 * 
 * Bridges the execution architecture with the existing conversation/responseEngine
 * to enable AI reasoning in swarms. This is a minimal integration that reuses
 * the existing infrastructure instead of reimplementing it.
 */
export class ConversationBridge {
    private readonly logger: Logger;
    private readonly promptService: PromptTemplateService;

    constructor(logger: Logger) {
        this.logger = logger;
        this.promptService = new PromptTemplateService(logger);
    }

    /**
     * Generate AI response for a swarm agent
     */
    async generateAgentResponse(
        agent: BotParticipant,
        swarmState: any,
        swarmConfig: any,
        userPrompt: string,
        chatId?: string,
    ): Promise<string> {
        try {
            // Build agent context using prompt template
            const systemPrompt = await this.promptService.buildAgentContext(
                agent,
                swarmState,
                swarmConfig,
            );

            // Create a minimal LLM completion task
            const task: LLMCompletionTask = {
                type: "LLM_COMPLETION",
                chatId: chatId || generatePK(),
                messageId: generatePK(),
                participantId: agent.id,
                userData: {
                    id: "system", // System-initiated
                    hasPremium: false,
                },
                messageContent: userPrompt,
                systemPrompt,
                // The responseEngine will handle loading tools based on agent role
            };

            // Use the existing completion service
            const response = await completionService.respond(task);
            
            // Extract the text response
            if (response && typeof response === "object" && "content" in response) {
                return response.content as string;
            }

            return "";
        } catch (error) {
            this.logger.error("[ConversationBridge] Failed to generate agent response", {
                agentId: agent.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Simple reasoning request without full conversation context
     */
    async reason(
        prompt: string,
        context?: Record<string, unknown>,
    ): Promise<string> {
        const agent: BotParticipant = {
            id: generatePK(),
            name: "Reasoning Agent",
            role: "analyzer",
            model: "gpt-4",
            meta: { temporary: true },
        };

        const swarmState = {
            state: "reasoning",
            ...context,
        };

        const swarmConfig = {
            goal: "Analyze and provide reasoning",
        };

        return this.generateAgentResponse(
            agent,
            swarmState,
            swarmConfig,
            prompt,
        );
    }
}