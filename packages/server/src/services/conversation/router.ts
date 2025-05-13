import { API_CREDITS_MULTIPLIER, BotSettings, LanguageModelResponseMode, SessionUser } from "@local/shared";
import { CustomError } from "../../events/error.js";
import { LlmServiceRegistry } from "../../tasks/llm/registry.js";
import { calculateMaxCredits } from "./credits.js";
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

export interface StreamOptions {
    /** Model name (provider‑specific). */
    model: string;
    /** Optional reference to continue a threaded conversation on provider side. */
    previous_response_id?: string;
    /** Message / input items per the Responses API. */
    input: Array<Record<string, unknown>>;
    /** JSON‑Schema tool list (built‑ins + custom). */
    tools: JsonSchema[];
    /** Allow model to emit several tool calls at once. */
    parallel_tool_calls?: boolean;
}

/** Limit chat responses to $0.50 for now */
// eslint-disable-next-line no-magic-numbers
const DEFAULT_MAX_RESPONSE_CREDITS = BigInt(50) * API_CREDITS_MULTIPLIER;

export abstract class LlmRouter {
    /**
     * Return the name of the preferred model for a given bot. Could consult user
     * subscription plan, provider health status, latency e.t.c.
     */
    abstract bestModelFor(botId: string): string;

    /** Built‑in tools to inject on **every** request (e.g. web_search). */
    abstract defaultTools(): JsonSchema[];

    /**
     * Stream model output in Responses‑API‑compatible chunks. The returned
     * AsyncIterable must yield events **in order**.
     */
    abstract stream(opts: StreamOptions): AsyncIterable<StreamEvent>;
}

export interface FallbackRouterCtorParams {
    /** user profile => plan, credit, localisation */
    userData: SessionUser;
    /** responding bot config (persona, preferred model) */
    botSettings: BotSettings;
    /** Whether to stream partial tokens */
    stream?: boolean;
    /** Force command style (used by your old context manager) */
    force?: boolean;
    /** Max cents ×1e6 allowed per call */
    maxCredits?: bigint;
    /** Response mode (you had chat/command etc.) */
    mode: LanguageModelResponseMode;
}

/**
 * Adapter: squeezes the old fallback helper behind the new `LlmRouter` iface.
 */
export class FallbackRouter extends LlmRouter {
    private static readonly RETRY_LIMIT = 3;

    constructor(private params: FallbackRouterCtorParams) {
        super();
    }

    bestModelFor(): string {
        // Delegated to the old helper – it chooses inside generateResponseWithFallback.
        return this.params.botSettings.model ?? "";
    }

    defaultTools(): JsonSchema[] {
        // Older completions flow had no built-ins; return empty array.
        return [];
    }

    async *stream(opts: StreamOptions): AsyncIterable<StreamEvent> {
        let attempts = 0;
        let accumulatedCost = 0;

        while (attempts++ < FallbackRouter.RETRY_LIMIT) {
            const registry = LlmServiceRegistry.get();
            const serviceId = registry.getBestService(this.params.botSettings.model);
            if (!serviceId) throw new CustomError("0252", "ServiceUnavailable", {});

            const service = registry.getService(serviceId);
            const model = service.getModel(this.params.botSettings.model);

            try {
                /* 1. input-safety & token window -------------------------------- */
                const raw = JSON.stringify(opts.input);
                const { cost: safetyCost, isSafe } = await service.safeInputCheck(raw);
                if (!isSafe) throw new CustomError("0605", "UnsafeContent", {});
                accumulatedCost += safetyCost;

                const inputTokens = service.estimateTokens({ aiModel: model, text: raw }).tokens;
                const tokenBudget = service.getMaxOutputTokensRestrained({
                    model,
                    maxCredits: calculateMaxCredits(
                        this.params.userData.credits,
                        this.params.maxCredits ?? DEFAULT_MAX_RESPONSE_CREDITS,
                        accumulatedCost,
                    ),
                    inputTokens,
                });
                if (tokenBudget !== null && tokenBudget <= 0)
                    throw new CustomError("0604", "CostLimitExceeded", {});

                /* 2. fire request ------------------------------------------------- */
                const stream = service.generateResponseStreaming({
                    messages: opts.input as any,        // ← legacy signature, fine
                    maxTokens: tokenBudget,
                    mode: this.params.mode,
                    model,
                    systemMessage: "",                 // already baked into input
                    userData: this.params.userData,
                });

                const responseId = `${serviceId}:${crypto.randomUUID()}`;
                for await (const chunk of stream) {
                    if (chunk.__type === "stream") {
                        yield { type: "message", content: chunk.message, responseId } satisfies MessageStreamEvent;
                    } else if (chunk.__type === "end") {
                        accumulatedCost += chunk.cost;
                        yield { type: "message", content: chunk.message, responseId, final: true } satisfies MessageStreamEvent;
                        yield { type: "done", cost: accumulatedCost, responseId } satisfies DoneStreamEvent;
                    } else if (chunk.__type === "error") {
                        throw new Error(chunk.message);
                    }
                }
                return;
            } catch (err) {
                const errType = service.getErrorType(err);
                registry.updateServiceState(serviceId, errType);
                if (attempts >= FallbackRouter.RETRY_LIMIT) throw err;
            }
        }
    }
}
