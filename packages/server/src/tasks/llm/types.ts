import { BotSettings, LanguageModelResponseMode, LlmServiceId, LlmTask, ModelInfo, SessionUser } from "@local/shared";
import { PreMapUserData } from "../../utils/chat.js";
import { ContextInfo } from "./context.js";
import { LlmServiceErrorType } from "./registry.js";

export type SimpleChatMessageData = {
    id: string;
    text: string;
}

export type LanguageModelMessage = Partial<ContextInfo> & {
    role: "user" | "assistant";
    content: string;
}
export type LanguageModelContext = {
    messages: LanguageModelMessage[];
    systemMessage: string;
}

export enum TokenEstimatorType {
    Default = "Default",
    Tiktoken = "Tiktoken",
}

export type EstimateTokensParams = {
    /** The requested model to base token logic on */
    aiModel: string;
    /** The text to estimate tokens for */
    text: string;
}
export type EstimateTokensResult = {
    /** The encoding used for the token estimation */
    encoding: string;
    /** The name of the token estimation model used (if requested one was invalid/incomplete) */
    estimationModel: TokenEstimatorType;
    /** The estimated amount of tokens calculated by this method/encoding pair */
    tokens: number;
}

export type GetConfigObjectParams = {
    botSettings: BotSettings,
    includeInitMessage: boolean,
    mode: LanguageModelResponseMode,
    userData: Pick<SessionUser, "languages">,
    task: LlmTask | `${LlmTask}` | undefined,
    force: boolean,
}
export type GenerateContextParams = {
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    contextInfo: ContextInfo[];
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    /**
     * The model to use for generating the response.
     */
    model: string;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    task: LlmTask | `${LlmTask}` | undefined;
    taskMessage?: string | null;
    userData: SessionUser;
}
export type GenerateResponseParams = {
    /**
     * Messages to include as context for the response. 
     * Typically the whole chat history tree, or as many as you can fit 
     * within the current token limit.
     */
    messages: LanguageModelMessage[];
    /**
     * The maximum number of tokens to output (meaning the input cost is not included). 
     * The model may (and most likely will) stop before reaching this limit.
     * If null, the service will decide what the maximum output should be.
     */
    maxTokens: number | null;
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    /**
     * The model to use for generating the response.
     */
    model: string;
    /**
     * The system message to include in the context. This is typically
     * the configuration for the bot and task instructions.
     */
    systemMessage: string;
    /**
     * Information about the user requesting the response.
     */
    userData: Pick<SessionUser, "languages" | "name">;
}
export type GenerateResponseResult = {
    /** How many attempts it took to generate a reponse */
    attempts: number;
    /** The generated message */
    message: string;
    /** How many API credits were used to generate the response */
    cost: number;
}
export type GenerateResponseStreamingResult = {
    __type: "stream" | "end" | "error";
    /**
     * The current stream of the message if __type is "stream" 
     * (so we can minimize the amount of data sent to the UI), 
     * or the full message if __type is "end"
     */
    message: string;
    /**
     * The accumulated cost of the response so far, or the total 
     * cost if __type is "end"
     */
    cost: number;
}

export type GetResponseCostParams = {
    model: string;
    usage: { input: number, output: number };
}

export type GetOutputTokenLimitParams = {
    maxCredits: bigint;
    model: string;
    inputTokens: number;
}

export type GetOutputTokenLimitResult = {
    cost: number;
    isSafe: boolean;
}

export interface LanguageModelService<GenerateNameType extends string> {
    /** Identifier for service */
    __id: LlmServiceId;
    /** Estimate the amount of tokens a string is */
    estimateTokens(params: EstimateTokensParams): EstimateTokensResult;
    /** Generates config for setting up bot persona and task instructions */
    getConfigObject(params: GetConfigObjectParams): Promise<Record<string, unknown>>;
    /** Generates context prompt from messages */
    generateContext(params: GenerateContextParams): Promise<LanguageModelContext>;
    /** Generate a message response, non-streaming */
    generateResponse(params: GenerateResponseParams): Promise<GenerateResponseResult>;
    /**  Generate a message repsonse, streaming */
    generateResponseStreaming(params: GenerateResponseParams): AsyncIterable<GenerateResponseStreamingResult>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** @returns Information about the available models */
    getModelInfo(): Record<GenerateNameType, ModelInfo>;
    /** @returns the max output window for the model */
    getMaxOutputTokens(requestedModel?: string | null): number;
    /**
     * Calculates the maximum number of tokens that can be output by the model 
     * to stay under the cost limit.
     * @param maxCredits The maximum cost (in cents * 1_000_000) that the response can have
     * @param model The model to use for the calculation
     * @param inputTokens The number of tokens in the input
     * @returns The maximum number of tokens that can be output, or 0 if the cost is too low 
     * (i.e. the input already exceeds the cost limit)
     */
    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number;
    /** 
     * Calculates the api credits used by the generation of an LLM response, 
     * based on the model's input token and output token costs (in cents per 1_000_000 tokens).
     * 
     * NOTE: Instead of using decimals, the cost is in cents multiplied by 1_000_000. This 
     * can be normalized in the UI so the number isn't overwhelming. 
     * 
     * @returns the total cost of the response
     */
    getResponseCost(params: GetResponseCostParams): number;
    /** @returns the estimation model and encoding for the model */
    getEstimationInfo(model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding">;
    /** Convert a preferred model to an available one */
    getModel(model?: string | null): GenerateNameType;
    /** Converts error received by service to a standardized type */
    getErrorType(error: unknown): LlmServiceErrorType;
    /** 
     * Checks if the input contains potentially harmful content 
     * @returns true if the input is safe, false otherwise
     */
    safeInputCheck(input: string): Promise<GetOutputTokenLimitResult>;
}
