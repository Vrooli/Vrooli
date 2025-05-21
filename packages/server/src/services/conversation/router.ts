import { API_CREDITS_MULTIPLIER } from "@local/shared";
import { randomUUID } from "crypto";
import { CustomError } from "../../events/error.js";
import { calculateMaxCredits } from "./credits.js";
import { AIServiceRegistry } from "./registry.js";
import { type ResponseStreamOptions } from "./services.js";
import { type JsonSchema } from "./types.js";

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

export type StreamOptions = Omit<ResponseStreamOptions, "maxTokens"> & {
    /** 
     * Max cents (* API_CREDITS_MULTIPLIER) allowed per call.
     */
    maxCredits?: bigint;
};

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
     * Orchestrates streaming responses from LLM services that implement the `AIService` interface.
     *
     * This method manages the process of sending a request to an LLM service and 
     * transforming its output into a standardized stream of router-level `StreamEvent` objects.
     * It includes a retry mechanism for transient service errors and expects services
     * to adhere to the `generateResponseStreaming` signature defined in `AIService`.
     *
     * The process involves several key steps:
     * 1. **Service Discovery and Initialization**: Identifies and retrieves the best available LLM service
     *    based on the requested model using `AIServiceRegistry`.
     * 2. **Token Budgeting**: 
     *    - Estimates the number of tokens in the input messages using `service.estimateTokens()`.
     *    - Calculates the maximum number of output tokens (`maxTokens`) the service is allowed to generate
     *      using `service.getMaxOutputTokensRestrained()`, considering user credit limits and input token count.
     * 3. **Service Call Preparation**: 
     *    - Constructs the `ResponseStreamOptions` (`svcOpts`) object for the target LLM service's 
     *      `generateResponseStreaming` method. This includes the original input messages (`opts.input`),
     *      tools, and the calculated `maxTokens`.
     *    - The underlying service (e.g., `OpenAIService`) is then responsible for:
     *        - Generating the final LLM-specific context from `svcOpts.input` (e.g., by calling its internal `generateContext` method,
     *          incorporating conversation history and a system prompt, especially if `previous_response_id` is not used).
     *        - Performing an input safety check (e.g., via `service.safeInputCheck()`).
     * 4. **Service Invocation**: 
     *    - Invokes `service.generateResponseStreaming(svcOpts)`.
     * 5. **Stream Processing and Event Yielding**:
     *    - Generates a unique `responseId` for the entire interaction.
     *    - Asynchronously iterates over the stream of `ServiceStreamEvent` items (e.g., text, function_call, done)
     *      returned by the service.
     *    - Maps each `ServiceStreamEvent` to a router `StreamEvent` by adding the `responseId`.
     *    - For the final "done" event, the `cost` received from the service (which should already include
     *      all relevant charges like safety check and model usage) is forwarded.
     *    - Yields each processed router `StreamEvent` to the caller.
     * 6. **Error Handling and Retries**:
     *    - Catches errors that occur during the service call or stream processing.
     *    - Updates the state of the service in the registry if an error occurs.
     *    - Retries the entire process up to `RETRY_LIMIT` times for recoverable errors.
     */
    async *stream(opts: StreamOptions): AsyncIterable<StreamEvent> {
        let attempts = 0;
        while (attempts++ < FallbackRouter.RETRY_LIMIT) {
            const registry = AIServiceRegistry.get();
            const serviceId = registry.getBestService(opts.model);
            if (!serviceId) throw new CustomError("0252", "ServiceUnavailable", {});

            const service = registry.getService(serviceId);

            try {
                // Compute token budget based on user credits and input size
                const safeMaxCredits = opts.maxCredits ?? DEFAULT_MAX_RESPONSE_CREDITS;
                const effectiveCredits = calculateMaxCredits(opts.userData.credits, safeMaxCredits, 0);
                // Estimate tokens for input
                const serializedInput = JSON.stringify(opts.input);
                const inputTokens = service.estimateTokens({ aiModel: opts.model, text: serializedInput }).tokens;
                const maxTokens = service.getMaxOutputTokensRestrained({ model: opts.model, maxCredits: effectiveCredits, inputTokens });
                // Build the service-level streaming options with token budget
                const svcOpts: ResponseStreamOptions = {
                    model: opts.model,
                    previous_response_id: opts.previous_response_id,
                    input: opts.input,
                    tools: opts.tools,
                    parallel_tool_calls: opts.parallel_tool_calls,
                    reasoningEffort: opts.reasoningEffort,
                    userData: opts.userData,
                    world: opts.world,
                    maxTokens,
                };

                const responseId = `${serviceId}:${randomUUID()}`;
                for await (const ev of service.generateResponseStreaming(svcOpts)) {
                    switch (ev.type) {
                        case "text":
                            yield { type: "message", content: ev.content, responseId } as MessageStreamEvent;
                            break;
                        case "function_call":
                            // ev has name, arguments, callId
                            yield {
                                type: "function_call",
                                name: ev.name,
                                arguments: ev.arguments,
                                callId: ev.callId,
                                responseId,
                            } as FunctionCallStreamEvent;
                            break;
                        case "reasoning":
                            yield { type: "reasoning", content: ev.content, responseId } as ReasoningStreamEvent;
                            break;
                        case "done":
                            yield { type: "done", cost: ev.cost, responseId } as DoneStreamEvent;
                            break;
                    }
                }
                return;
            } catch (err: unknown) {
                const errType = service.getErrorType(err);
                registry.updateServiceState(serviceId, errType);
                if (attempts >= FallbackRouter.RETRY_LIMIT) throw err;
            }
        }
    }
}
