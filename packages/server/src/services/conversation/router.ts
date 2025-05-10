import { BotSettings, LanguageModelResponseMode, SessionUser } from "@local/shared";
import { LlmServiceRegistry } from "../../tasks/llm/registry.js";
import { JsonSchema, LLM, StreamOptions } from "./types.js";

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
    abstract stream(opts: StreamOptions): AsyncIterable<LLM.StreamEvent>;
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

    async *stream(opts: StreamOptions): AsyncIterable<LLM.StreamEvent> {
        let attempts = 0;
        let accumulatedCost = 0;

        while (attempts < RETRY_LIMIT) {
            attempts++;

            /* 1️⃣ choose a live service */
            const registry = LlmServiceRegistry.get();
            const serviceId = registry.getBestService(this.opts.botConfig.model);
            if (!serviceId) throw new CustomError("0252", "ServiceUnavailable", {});

            const service = registry.getService(serviceId);
            const model = service.getModel(this.opts.botConfig.model);

            try {
                /* 2️⃣ safe‑input check */
                const rawInput = JSON.stringify(streamOpts.input);
                const safety = await service.safeInputCheck(rawInput);
                accumulatedCost += safety.cost;
                if (!safety.isSafe) throw new CustomError("0605", "UnsafeContent", {});

                /* 3️⃣ compute token limits */
                const inputTokens = service.estimateTokens({ aiModel: model, text: rawInput }).tokens;
                const maxOutputCred = calculateMaxCredits(
                    this.opts.userData.credits,
                    this.opts.maxCredits ?? DEFAULT_MAX_RESPONSE_CREDITS,
                    accumulatedCost
                );
                const maxOutTokens = service.getMaxOutputTokensRestrained({
                    model,
                    maxCredits: maxOutputCred,
                    inputTokens,
                });
                if (maxOutTokens !== null && maxOutTokens <= 0) {
                    throw new CustomError("0604", "CostLimitExceeded", {});
                }

                /* 4️⃣ generate */
                if (this.opts.stream ?? true) {
                    const response = service.generateResponseStreaming({
                        messages: streamOpts.input as any,
                        maxTokens: maxOutTokens,
                        mode: this.opts.mode,
                        model,
                        systemMessage: "", // already inside input array
                        userData: this.opts.userData,
                    });

                    for await (const chunk of response) {
                        if (chunk.__type === "stream") {
                            yield {
                                type: "message",
                                content: chunk.message,
                                response_id: serviceId + "-stream",
                            } as LLM.StreamEvent;
                        } else if (chunk.__type === "end") {
                            yield {
                                type: "message",
                                content: chunk.message,
                                response_id: serviceId + "-end",
                            } as LLM.StreamEvent;
                            return; // turn finished
                        } else if (chunk.__type === "error") {
                            throw new Error(chunk.message);
                        }
                    }
                } else {
                    const { message } = await service.generateResponse({
                        messages: streamOpts.input as any,
                        maxTokens: maxOutTokens,
                        model,
                        mode: this.opts.mode,
                        systemMessage: "",
                        userData: this.opts.userData,
                    });
                    yield {
                        type: "message",
                        content: message,
                        response_id: serviceId + "-single",
                    } as LLM.StreamEvent;
                }
                return; // success – stop retry loop
            } catch (err) {
                // unwrap registry failure handling
                const errorType = service.getErrorType?.(err) ?? LlmServiceErrorType.ApiError;
                registry.updateServiceState(serviceId.toString(), errorType);
                logger.warning("Router retry via", { serviceId, attempts, errorType });
                if (attempts >= RETRY_LIMIT) throw err; // surface after retries
            }
        }
    }
}

/* ------------------------------------------------------------------
 * HOW TO GRADUALLY UPGRADE TO FULL RESPONSES‑API SUPPORT
 * ------------------------------------------------------------------
 * 1) Upgrade one LanguageModelService (eg OpenAIService) to expose a new
 *    generateResponsesStream() that yields provider delta JSON.
 * 2) Add feature‑detection into FallbackRouter: if chosen service supports the
 *    new method, shortcut to that path and call `normaliseProviderChunk()` –
 *    exactly like HttpResponsesRouter does – then forward "function_call"
 *    events to ConversationLoop.
 * 3) Keep legacy path for services not yet migrated.
 * ------------------------------------------------------------------ */
