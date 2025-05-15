import { API_CREDITS_MULTIPLIER } from "@local/shared";
import { CustomError } from "../../events/error.js";
import { LanguageModelMessage, ServiceStreamOptions } from "../../tasks/llm/types.js";
import { ContextInfo } from "./contextBuilder.js";
import { calculateMaxCredits } from "./credits.js";
import { LlmServiceRegistry } from "./registry.js";
import { ResponseStreamOptions } from "./services.js";
import { JsonSchema } from "./types.js";

export type MessageStreamEvent = {
    type: "message";
    /** Incremental content if streaming, or full message if done */
    content: string;
    /** Unique identifier for the response */
    responseId: string;
    /** True if this is the final chunk of the message */
    final?: boolean;
};
export type FunctionCallStreamEvent = {
    type: "function_call";
    name: string;
    arguments: unknown;
    callId: string;
    responseId: string;
};

export type ReasoningStreamEvent = {
    type: "reasoning";
    content: string;
    responseId: string;
};

/** Emitted once after the final message */
export type DoneStreamEvent = {
    type: "done";
    /** μ-credits already *including* safety check */
    cost: number;
    /** Unique identifier for the response */
    responseId: string;
};

export type StreamEvent =
    | MessageStreamEvent
    | FunctionCallStreamEvent
    | ReasoningStreamEvent
    | DoneStreamEvent;

export type StreamOptions = ResponseStreamOptions;

/** Limit chat responses to $0.50 for now */
// eslint-disable-next-line no-magic-numbers
const DEFAULT_MAX_RESPONSE_CREDITS = BigInt(50) * API_CREDITS_MULTIPLIER;

export abstract class LlmRouter {

    /** Built‑in tools to inject on **every** request (e.g. web_search). */
    abstract defaultTools(): JsonSchema[];

    /**
     * Stream model output in Responses‑API‑compatible chunks. The returned
     * AsyncIterable must yield events **in order**.
     */
    abstract stream(opts: StreamOptions): AsyncIterable<StreamEvent>;
}

/**
 * Adapter: squeezes the old fallback helper behind the new `LlmRouter` iface.
 */
export class FallbackRouter extends LlmRouter {
    private static readonly RETRY_LIMIT = 3;

    constructor() {
        super();
    }

    defaultTools(): JsonSchema[] {
        // Older completions flow had no built-ins; return empty array.
        return [];
    }

    /**
     * Orchestrates streaming responses from LLM services that implement the `LanguageModelService` interface.
     *
     * This method manages the process of sending a request to an LLM service and 
     * transforming its output into a standardized stream of `StreamEvent` objects.
     * It includes a retry mechanism for transient service errors and expects services
     * to adhere to the modernized `generateResponseStreaming` signature defined in `LanguageModelService`.
     *
     * The process involves several key steps:
     * 1. **Service Discovery and Initialization**: Identifies and retrieves the best available LLM service
     *    based on the requested model.
     * 2. **Context Preparation**: 
     *    - Maps the raw `opts.input` to a `ContextInfo[]` structure.
     *    - Calls the service's `generateContext` method to process this input, 
     *      which typically involves incorporating conversation history and a system prompt.
     *    - Finalizes the list of messages to be sent, ensuring the system message is correctly prioritized 
     *      (from `generateContext` or `botSettings`).
     * 3. **Input Safety and Token Budgeting**:
     *    - Performs a safety check on the content of the finalized messages.
     *    - Estimates the number of tokens in the input messages.
     *    - Calculates the maximum number of output tokens (`tokenBudget`) the service is allowed to generate,
     *      considering user credit limits, previously accumulated costs (e.g., safety check cost), and input token count.
     * 4. **Service Call**: 
     *    - Constructs the `ServiceStreamOptions` object tailored for the target LLM service's 
     *      `generateResponseStreaming` method (as defined in the `LanguageModelService` interface),
     *      including the finalized messages, tools, and token budget.
     *    - Invokes `service.generateResponseStreaming`.
     * 5. **Stream Processing and Event Yielding**:
     *    - Generates a unique `responseId` for the entire interaction.
     *    - Asynchronously iterates over the stream of events (`Omit<StreamEvent, 'responseId'>`) returned by the service.
     *    - Adds the `responseId` to each event.
     *    - For the final "done" event, it aggregates the cost from the safety check with the cost reported by the service.
     *    - Yields each processed `StreamEvent` to the caller.
     * 6. **Error Handling and Retries**:
     *    - Catches errors that occur during the service call or stream processing.
     *    - Updates the state of the service if an error occurs (e.g., marking it as potentially unavailable).
     *    - Retries the entire process up to `RETRY_LIMIT` times for recoverable errors.
     */
    async *stream(opts: StreamOptions): AsyncIterable<StreamEvent> {
        let attempts = 0;
        let accumulatedCost = 0;

        while (attempts++ < FallbackRouter.RETRY_LIMIT) {
            // Step 1: Service Discovery and Initialization
            const registry = LlmServiceRegistry.get();
            const serviceId = registry.getBestService(opts.model);
            if (!serviceId) throw new CustomError("0252", "ServiceUnavailable", {});

            const service = registry.getService(serviceId);
            const model = service.getModel(opts.model);

            try {
                // Step 2: Context Preparation
                const contextInfoForGenerator: ContextInfo[] = (opts.input as any[]).map((item: any): ContextInfo => {
                    if (item && typeof item.__type === "string") {
                        return item as ContextInfo;
                    }
                    // Placeholder: This mapping needs to be robust based on actual structure of opts.input items.
                    return { __type: "text", text: JSON.stringify(item), tokenSize: 0, userId: null };
                });

                const anyBotSettings = opts.botSettings as any;

                const { messages: contextMessages, systemMessage: generatedSystemMessage } = await service.generateContext({
                    model: opts.model,
                    force: opts.force ?? false,
                    contextInfo: contextInfoForGenerator,
                    mode: opts.mode,
                    respondingBotId: anyBotSettings?.id ?? "unknown-bot",
                    respondingBotConfig: opts.botSettings,
                    userData: opts.userData,
                    task: undefined,
                    taskMessage: undefined,
                });

                const finalMessages: LanguageModelMessage[] = [];
                if (generatedSystemMessage && !contextMessages.some(m => m.role === "system")) {
                    finalMessages.push({ role: "system", content: generatedSystemMessage });
                }
                finalMessages.push(...contextMessages);

                const botSystemMessage = anyBotSettings?.persona?.system_message ?? anyBotSettings?.meta?.systemPrompt;
                if (!finalMessages.some(m => m.role === "system") && botSystemMessage) {
                    finalMessages.unshift({ role: "system", content: botSystemMessage });
                }

                // Step 3: Input Safety and Token Budgeting
                const rawInputForSafetyCheck = JSON.stringify(finalMessages);
                const { cost: safetyCost, isSafe } = await service.safeInputCheck(rawInputForSafetyCheck);
                if (!isSafe) throw new CustomError("0605", "UnsafeContent", {});
                accumulatedCost += safetyCost;

                const inputTokens = service.estimateTokens({ aiModel: model, text: rawInputForSafetyCheck }).tokens;
                const tokenBudget = service.getMaxOutputTokensRestrained({
                    model,
                    maxCredits: calculateMaxCredits(
                        opts.userData.credits,
                        opts.maxCredits ?? DEFAULT_MAX_RESPONSE_CREDITS,
                        accumulatedCost,
                    ),
                    inputTokens,
                });
                if (tokenBudget !== null && tokenBudget <= 0)
                    throw new CustomError("0604", "CostLimitExceeded", {});

                // Step 4: Service Call
                const serviceParams: ServiceStreamOptions = {
                    model: opts.model,
                    previous_response_id: opts.previous_response_id,
                    messages: finalMessages,
                    tools: opts.tools,
                    parallel_tool_calls: opts.parallel_tool_calls,
                    userData: opts.userData,
                    botSettings: opts.botSettings,
                    mode: opts.mode,
                    maxOutputTokens: tokenBudget,
                };

                const streamResponse = service.generateResponseStreaming(serviceParams);

                // Step 5: Stream Processing and Event Yielding
                const responseId = `${serviceId}:${crypto.randomUUID()}`;
                for await (const chunk of streamResponse) {
                    if (chunk.type === "done") {
                        const doneChunk = chunk as Omit<DoneStreamEvent, "responseId">;
                        yield { ...doneChunk, cost: accumulatedCost + doneChunk.cost, responseId } satisfies DoneStreamEvent;
                    } else {
                        yield { ...chunk, responseId } as StreamEvent;
                    }
                }
                return;
            } catch (err) {
                // Step 6: Error Handling and Retries
                const errType = service.getErrorType(err);
                registry.updateServiceState(serviceId, errType);
                if (attempts >= FallbackRouter.RETRY_LIMIT) throw err;
            }
        }
    }
}
