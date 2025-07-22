/* c8 ignore start */
import { type TranslationKeyService } from "../types.d.js";

export enum AIServiceName {
    LocalOllama = "LocalOllama",
    CloudflareGateway = "CloudflareGateway",
    OpenRouter = "OpenRouter",
    ClaudeCode = "ClaudeCode",
}

/** All possible features that a model can support */
export enum ModelFeature {
    CodeInterpreter = "CodeInterpreter",
    FileSearch = "FileSearch",
    FunctionCalling = "FunctionCalling",
    Vision = "Vision",
    ImageGeneration = "ImageGeneration",
    Transcription = "Transcription",
}

export type ImageCostDetail = {
    quality: "standard" | "hd";
    resolution: string; // e.g., "1024x1024", "1024x1792"
    costPerImage: number; // in cents
};

export type ModelFeatureInfo =
    | { type: "image_generation"; modelName: "DALL-E 3" | "DALL-E 2"; costs: ImageCostDetail[]; }
    | { type: "transcription"; modelName: "Whisper"; costPerMinute: number; } // in cents
    | { type: "vision"; notes?: string; } // Details on how vision capabilities are priced
    | { type: "generic"; notes?: string; details?: Record<string, any> }; // For general features

/**
 * Information to describe the cost, limits, and capabilities of an AI model
 */
export type ModelInfo = {
    /** True if the model can be selected */
    enabled: boolean;
    /** The models' name, which will be displayed to the user */
    name: TranslationKeyService;
    /** A short description for the model */
    descriptionShort: TranslationKeyService;
    /** Cost in cents per API_CREDITS_MULTIPLIER (typically 1M) input tokens */
    inputCost: number;
    /** Cost in cents per API_CREDITS_MULTIPLIER (typically 1M) output tokens */
    outputCost: number;
    /** Max context window */
    contextWindow: number;
    /** Max output tokens */
    maxOutputTokens: number;
    /** Features supported by the model, and their metadata */
    features: { [key in ModelFeature]?: ModelFeatureInfo };
    /** Whether the model has specialized reasoning capabilities */
    supportsReasoning?: boolean;
};

/**
 * ModelInfo variant for third-party providers that may not have translation keys
 */
export type ThirdPartyModelInfo = Omit<ModelInfo, "name" | "descriptionShort"> & {
    /** The models' name as a string literal for third-party providers */
    name: string;
    /** A short description for the model as a string literal */
    descriptionShort: string;
};

/**
 * Information about the models and behavior of an AI service
 */
type AIServiceInfo<Models extends string> = {
    /** The default model to use for the service */
    defaultModel: Models;
    /** True if the service can be selected */
    enabled: boolean;
    /** The service's name, which will be displayed to the user */
    name: string;
    fallbackMaxTokens: number;
    models: Record<Models, ModelInfo>;
    /** Order to display the models in the UI */
    displayOrder: Models[];
}

export type LlmServiceModel = LocalOllamaModel | CloudflareGatewayModel | OpenRouterModel | ClaudeCodeModel;

/**
 * All available services
 */
export enum LlmServiceId {
    LocalOllama = "LocalOllama",
    CloudflareGateway = "CloudflareGateway",
    OpenRouter = "OpenRouter",
    ClaudeCode = "ClaudeCode",
}

/**
 * Information about all available AI services
 */
export type AIServicesInfo = {
    defaultService: LlmServiceId;
    services: {
        [AIServiceName.LocalOllama]: AIServiceInfo<LocalOllamaModel>,
        [AIServiceName.CloudflareGateway]: AIServiceInfo<CloudflareGatewayModel>,
        [AIServiceName.OpenRouter]: AIServiceInfo<OpenRouterModel>,
        [AIServiceName.ClaudeCode]: AIServiceInfo<ClaudeCodeModel>,
    },
    fallbacks: Record<LlmServiceModel, LlmServiceModel[]>,
};

// Resources:
// - https://ollama.com/library
export enum LocalOllamaModel {
    // Dynamic models - populated at runtime from Ollama API
    Llama3_1_8B = "llama3.1:8b",
    Llama3_1_70B = "llama3.1:70b",
    CodeLlama_13B = "codellama:13b",
    Mistral_7B = "mistral:7b",
    Gemma2_9B = "gemma2:9b",
    Phi3_Medium = "phi3:medium",
    Dynamic = "dynamic", // Placeholder for runtime-discovered models
}

export const localOllamaServiceInfo: AIServiceInfo<LocalOllamaModel> = {
    defaultModel: LocalOllamaModel.Llama3_1_8B,
    enabled: true,
    name: "Local Ollama",
    fallbackMaxTokens: 4_096,
    models: {
        [LocalOllamaModel.Llama3_1_8B]: {
            enabled: true,
            name: "Llama_3_1_8B_Name",
            descriptionShort: "Llama_3_1_8B_Description_Short",
            inputCost: 0.001,  // Nominal cost for resource tracking
            outputCost: 0.001,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Basic function calling support" },
            },
            supportsReasoning: false,
        },
        [LocalOllamaModel.Llama3_1_70B]: {
            enabled: true,
            name: "Llama_3_1_70B_Name",
            descriptionShort: "Llama_3_1_70B_Description_Short",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Advanced function calling support" },
            },
            supportsReasoning: false,
        },
        [LocalOllamaModel.CodeLlama_13B]: {
            enabled: true,
            name: "CodeLlama_13B_Name",
            descriptionShort: "CodeLlama_13B_Description_Short",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 16_384,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Specialized for code generation" },
            },
            supportsReasoning: false,
        },
        [LocalOllamaModel.Mistral_7B]: {
            enabled: true,
            name: "Mistral_7B_Name",
            descriptionShort: "Mistral_7B_Description_Short",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [LocalOllamaModel.Gemma2_9B]: {
            enabled: true,
            name: "Gemma2_9B_Name",
            descriptionShort: "Gemma2_9B_Description_Short",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [LocalOllamaModel.Phi3_Medium]: {
            enabled: true,
            name: "Phi3_Medium_Name",
            descriptionShort: "Phi3_Medium_Description_Short",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [LocalOllamaModel.Dynamic]: {
            enabled: true,
            name: "Dynamic_Ollama_Model",
            descriptionShort: "Runtime_Discovered_Model",
            inputCost: 0.001,
            outputCost: 0.001,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
    },
    displayOrder: [
        LocalOllamaModel.Llama3_1_8B,
        LocalOllamaModel.Llama3_1_70B,
        LocalOllamaModel.CodeLlama_13B,
        LocalOllamaModel.Mistral_7B,
        LocalOllamaModel.Gemma2_9B,
        LocalOllamaModel.Phi3_Medium,
        LocalOllamaModel.Dynamic,
    ],
};

// Resources:
// - https://developers.cloudflare.com/ai/models/
export enum CloudflareGatewayModel {
    // OpenAI models via Cloudflare
    GPT4o = "@cf/openai/gpt-4o",
    GPT4o_Mini = "@cf/openai/gpt-4o-mini",
    GPT4_Turbo = "@cf/openai/gpt-4-turbo",
    GPT3_5_Turbo = "@cf/openai/gpt-3.5-turbo",

    // Anthropic models via Cloudflare
    Claude3_Sonnet = "@cf/anthropic/claude-3-sonnet",
    Claude3_Haiku = "@cf/anthropic/claude-3-haiku",
    Claude3_5_Sonnet = "@cf/anthropic/claude-3-5-sonnet",

    // Meta models
    Llama3_8B = "@cf/meta/llama-3-8b-instruct",
    Llama3_70B = "@cf/meta/llama-3-70b-instruct",

    // Mistral models
    Mistral_7B = "@cf/mistral/mistral-7b-instruct",
    Mistral_8x7B = "@cf/mistral/mixtral-8x7b-instruct",

    // Microsoft models
    Phi2 = "@cf/microsoft/phi-2",

    // Google models
    Gemma_7B = "@cf/google/gemma-7b-it",
}

export const cloudflareGatewayServiceInfo: AIServiceInfo<CloudflareGatewayModel> = {
    defaultModel: CloudflareGatewayModel.GPT4o_Mini,
    enabled: true,
    name: "Cloudflare Gateway",
    fallbackMaxTokens: 4_096,
    models: {
        [CloudflareGatewayModel.GPT4o]: {
            enabled: true,
            name: "CF_GPT4o_Name",
            descriptionShort: "CF_GPT4o_Description_Short",
            inputCost: 250,  // Cloudflare Gateway pricing
            outputCost: 1000,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.GPT4o_Mini]: {
            enabled: true,
            name: "CF_GPT4o_Mini_Name",
            descriptionShort: "CF_GPT4o_Mini_Description_Short",
            inputCost: 50,
            outputCost: 150,
            contextWindow: 128_000,
            maxOutputTokens: 16_384,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Claude3_5_Sonnet]: {
            enabled: true,
            name: "CF_Claude3_5_Sonnet_Name",
            descriptionShort: "CF_Claude3_5_Sonnet_Description_Short",
            inputCost: 300,
            outputCost: 1500,
            contextWindow: 200_000,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Claude3_Haiku]: {
            enabled: true,
            name: "CF_Claude3_Haiku_Name",
            descriptionShort: "CF_Claude3_Haiku_Description_Short",
            inputCost: 25,
            outputCost: 125,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Llama3_8B]: {
            enabled: true,
            name: "CF_Llama3_8B_Name",
            descriptionShort: "CF_Llama3_8B_Description_Short",
            inputCost: 10,
            outputCost: 10,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Llama3_70B]: {
            enabled: true,
            name: "CF_Llama3_70B_Name",
            descriptionShort: "CF_Llama3_70B_Description_Short",
            inputCost: 50,
            outputCost: 50,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.GPT4_Turbo]: {
            enabled: true,
            name: "CF_GPT4_Turbo_Name",
            descriptionShort: "CF_GPT4_Turbo_Description_Short",
            inputCost: 1000,
            outputCost: 3000,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.GPT3_5_Turbo]: {
            enabled: true,
            name: "CF_GPT3_5_Turbo_Name",
            descriptionShort: "CF_GPT3_5_Turbo_Description_Short",
            inputCost: 50,
            outputCost: 150,
            contextWindow: 16_385,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Claude3_Sonnet]: {
            enabled: true,
            name: "CF_Claude3_Sonnet_Name",
            descriptionShort: "CF_Claude3_Sonnet_Description_Short",
            inputCost: 300,
            outputCost: 1500,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Mistral_7B]: {
            enabled: true,
            name: "CF_Mistral_7B_Name",
            descriptionShort: "CF_Mistral_7B_Description_Short",
            inputCost: 25,
            outputCost: 25,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Mistral_8x7B]: {
            enabled: true,
            name: "CF_Mistral_8x7B_Name",
            descriptionShort: "CF_Mistral_8x7B_Description_Short",
            inputCost: 50,
            outputCost: 50,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Phi2]: {
            enabled: true,
            name: "CF_Phi2_Name",
            descriptionShort: "CF_Phi2_Description_Short",
            inputCost: 10,
            outputCost: 10,
            contextWindow: 2_048,
            maxOutputTokens: 1_024,
            features: {},
            supportsReasoning: false,
        },
        [CloudflareGatewayModel.Gemma_7B]: {
            enabled: true,
            name: "CF_Gemma_7B_Name",
            descriptionShort: "CF_Gemma_7B_Description_Short",
            inputCost: 15,
            outputCost: 15,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
    },
    displayOrder: [
        CloudflareGatewayModel.GPT4o_Mini,
        CloudflareGatewayModel.GPT4o,
        CloudflareGatewayModel.Claude3_5_Sonnet,
        CloudflareGatewayModel.Claude3_Haiku,
        CloudflareGatewayModel.Llama3_8B,
        CloudflareGatewayModel.Llama3_70B,
        CloudflareGatewayModel.GPT4_Turbo,
        CloudflareGatewayModel.GPT3_5_Turbo,
        CloudflareGatewayModel.Claude3_Sonnet,
        CloudflareGatewayModel.Mistral_7B,
        CloudflareGatewayModel.Mistral_8x7B,
        CloudflareGatewayModel.Phi2,
        CloudflareGatewayModel.Gemma_7B,
    ],
};

// Resources:
// - https://openrouter.ai/models
export enum OpenRouterModel {
    // OpenAI models
    GPT4o = "openai/gpt-4o",
    GPT4o_Mini = "openai/gpt-4o-mini",
    GPT4_Turbo = "openai/gpt-4-turbo",
    GPT4 = "openai/gpt-4",
    GPT3_5_Turbo = "openai/gpt-3.5-turbo",
    o1_Preview = "openai/o1-preview",
    o1_Mini = "openai/o1-mini",

    // Anthropic models
    Claude3_5_Sonnet = "anthropic/claude-3.5-sonnet",
    Claude3_Opus = "anthropic/claude-3-opus",
    Claude3_Sonnet = "anthropic/claude-3-sonnet",
    Claude3_Haiku = "anthropic/claude-3-haiku",

    // Google models
    Gemini_1_5_Pro = "google/gemini-pro-1.5",
    Gemini_1_5_Flash = "google/gemini-flash-1.5",
    Gemini_Pro = "google/gemini-pro",

    // Meta models
    Llama3_1_405B = "meta-llama/llama-3.1-405b-instruct",
    Llama3_1_70B = "meta-llama/llama-3.1-70b-instruct",
    Llama3_1_8B = "meta-llama/llama-3.1-8b-instruct",
    Llama3_70B = "meta-llama/llama-3-70b-instruct",
    Llama3_8B = "meta-llama/llama-3-8b-instruct",

    // Mistral models
    Mistral_Large2 = "mistralai/mistral-large-2407",
    Mistral_Nemo = "mistralai/mistral-nemo",
    Mistral_7B = "mistralai/mistral-7b-instruct",
    Codestral = "mistralai/codestral-mamba",

    // Other popular models
    Mixtral_8x7B = "mistralai/mixtral-8x7b-instruct",
    Mixtral_8x22B = "mistralai/mixtral-8x22b-instruct",
    Qwen2_72B = "qwen/qwen-2-72b-instruct",
    DeepSeek_Coder = "deepseek/deepseek-coder",
    CodeLlama_34B = "meta-llama/codellama-34b-instruct",
}

export const openRouterServiceInfo: AIServiceInfo<OpenRouterModel> = {
    defaultModel: OpenRouterModel.GPT4o_Mini,
    enabled: true,
    name: "OpenRouter",
    fallbackMaxTokens: 4_096,
    models: {
        [OpenRouterModel.GPT4o]: {
            enabled: true,
            name: "OR_GPT4o_Name",
            descriptionShort: "OR_GPT4o_Description_Short",
            inputCost: 250,  // OpenRouter competitive pricing
            outputCost: 1000,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.GPT4o_Mini]: {
            enabled: true,
            name: "OR_GPT4o_Mini_Name",
            descriptionShort: "OR_GPT4o_Mini_Description_Short",
            inputCost: 15,
            outputCost: 60,
            contextWindow: 128_000,
            maxOutputTokens: 16_384,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Claude3_5_Sonnet]: {
            enabled: true,
            name: "OR_Claude3_5_Sonnet_Name",
            descriptionShort: "OR_Claude3_5_Sonnet_Description_Short",
            inputCost: 300,
            outputCost: 1500,
            contextWindow: 200_000,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Claude3_Opus]: {
            enabled: true,
            name: "OR_Claude3_Opus_Name",
            descriptionShort: "OR_Claude3_Opus_Description_Short",
            inputCost: 1500,
            outputCost: 7500,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Llama3_1_405B]: {
            enabled: true,
            name: "OR_Llama3_1_405B_Name",
            descriptionShort: "OR_Llama3_1_405B_Description_Short",
            inputCost: 300,
            outputCost: 300,
            contextWindow: 131_072,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Llama3_1_70B]: {
            enabled: true,
            name: "OR_Llama3_1_70B_Name",
            descriptionShort: "OR_Llama3_1_70B_Description_Short",
            inputCost: 40,
            outputCost: 40,
            contextWindow: 131_072,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Llama3_1_8B]: {
            enabled: true,
            name: "OR_Llama3_1_8B_Name",
            descriptionShort: "OR_Llama3_1_8B_Description_Short",
            inputCost: 6,
            outputCost: 6,
            contextWindow: 131_072,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Gemini_1_5_Pro]: {
            enabled: true,
            name: "OR_Gemini_1_5_Pro_Name",
            descriptionShort: "OR_Gemini_1_5_Pro_Description_Short",
            inputCost: 125,
            outputCost: 375,
            contextWindow: 2_097_152,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Mistral_Large2]: {
            enabled: true,
            name: "OR_Mistral_Large2_Name",
            descriptionShort: "OR_Mistral_Large2_Description_Short",
            inputCost: 300,
            outputCost: 900,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Codestral]: {
            enabled: true,
            name: "OR_Codestral_Name",
            descriptionShort: "OR_Codestral_Description_Short",
            inputCost: 25,
            outputCost: 25,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Mixtral_8x7B]: {
            enabled: true,
            name: "OR_Mixtral_8x7B_Name",
            descriptionShort: "OR_Mixtral_8x7B_Description_Short",
            inputCost: 24,
            outputCost: 24,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.GPT4_Turbo]: {
            enabled: true,
            name: "OR_GPT4_Turbo_Name",
            descriptionShort: "OR_GPT4_Turbo_Description_Short",
            inputCost: 1000,
            outputCost: 3000,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.GPT4]: {
            enabled: true,
            name: "OR_GPT4_Name",
            descriptionShort: "OR_GPT4_Description_Short",
            inputCost: 3000,
            outputCost: 6000,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.GPT3_5_Turbo]: {
            enabled: true,
            name: "OR_GPT3_5_Turbo_Name",
            descriptionShort: "OR_GPT3_5_Turbo_Description_Short",
            inputCost: 50,
            outputCost: 150,
            contextWindow: 16_385,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.o1_Preview]: {
            enabled: true,
            name: "OR_o1_Preview_Name",
            descriptionShort: "OR_o1_Preview_Description_Short",
            inputCost: 1500,
            outputCost: 6000,
            contextWindow: 128_000,
            maxOutputTokens: 32_768,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
            },
            supportsReasoning: true,
        },
        [OpenRouterModel.o1_Mini]: {
            enabled: true,
            name: "OR_o1_Mini_Name",
            descriptionShort: "OR_o1_Mini_Description_Short",
            inputCost: 300,
            outputCost: 1200,
            contextWindow: 128_000,
            maxOutputTokens: 65_536,
            features: {
                [ModelFeature.FunctionCalling]: { type: "generic" },
            },
            supportsReasoning: true,
        },
        [OpenRouterModel.Claude3_Sonnet]: {
            enabled: true,
            name: "OR_Claude3_Sonnet_Name",
            descriptionShort: "OR_Claude3_Sonnet_Description_Short",
            inputCost: 300,
            outputCost: 1500,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Claude3_Haiku]: {
            enabled: true,
            name: "OR_Claude3_Haiku_Name",
            descriptionShort: "OR_Claude3_Haiku_Description_Short",
            inputCost: 25,
            outputCost: 125,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Llama3_70B]: {
            enabled: true,
            name: "OR_Llama3_70B_Name",
            descriptionShort: "OR_Llama3_70B_Description_Short",
            inputCost: 52,
            outputCost: 75,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Llama3_8B]: {
            enabled: true,
            name: "OR_Llama3_8B_Name",
            descriptionShort: "OR_Llama3_8B_Description_Short",
            inputCost: 6,
            outputCost: 6,
            contextWindow: 8_192,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Gemini_1_5_Flash]: {
            enabled: true,
            name: "OR_Gemini_1_5_Flash_Name",
            descriptionShort: "OR_Gemini_1_5_Flash_Description_Short",
            inputCost: 7,
            outputCost: 21,
            contextWindow: 1_048_576,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Gemini_Pro]: {
            enabled: true,
            name: "OR_Gemini_Pro_Name",
            descriptionShort: "OR_Gemini_Pro_Description_Short",
            inputCost: 12,
            outputCost: 37,
            contextWindow: 30_720,
            maxOutputTokens: 2_048,
            features: {
                [ModelFeature.Vision]: { type: "vision" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.Mistral_Nemo]: {
            enabled: true,
            name: "OR_Mistral_Nemo_Name",
            descriptionShort: "OR_Mistral_Nemo_Description_Short",
            inputCost: 18,
            outputCost: 18,
            contextWindow: 128_000,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Mistral_7B]: {
            enabled: true,
            name: "OR_Mistral_7B_Name",
            descriptionShort: "OR_Mistral_7B_Description_Short",
            inputCost: 6,
            outputCost: 6,
            contextWindow: 32_768,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Mixtral_8x22B]: {
            enabled: true,
            name: "OR_Mixtral_8x22B_Name",
            descriptionShort: "OR_Mixtral_8x22B_Description_Short",
            inputCost: 65,
            outputCost: 65,
            contextWindow: 65_536,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.Qwen2_72B]: {
            enabled: true,
            name: "OR_Qwen2_72B_Name",
            descriptionShort: "OR_Qwen2_72B_Description_Short",
            inputCost: 56,
            outputCost: 77,
            contextWindow: 131_072,
            maxOutputTokens: 4_096,
            features: {},
            supportsReasoning: false,
        },
        [OpenRouterModel.DeepSeek_Coder]: {
            enabled: true,
            name: "OR_DeepSeek_Coder_Name",
            descriptionShort: "OR_DeepSeek_Coder_Description_Short",
            inputCost: 14,
            outputCost: 28,
            contextWindow: 16_384,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic" },
            },
            supportsReasoning: false,
        },
        [OpenRouterModel.CodeLlama_34B]: {
            enabled: true,
            name: "OR_CodeLlama_34B_Name",
            descriptionShort: "OR_CodeLlama_34B_Description_Short",
            inputCost: 80,
            outputCost: 80,
            contextWindow: 16_384,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic" },
            },
            supportsReasoning: false,
        },
    },
    displayOrder: [
        OpenRouterModel.GPT4o_Mini,
        OpenRouterModel.GPT4o,
        OpenRouterModel.Claude3_5_Sonnet,
        OpenRouterModel.Claude3_Haiku,
        OpenRouterModel.Llama3_1_8B,
        OpenRouterModel.Llama3_1_70B,
        OpenRouterModel.Llama3_1_405B,
        OpenRouterModel.Gemini_1_5_Flash,
        OpenRouterModel.Gemini_1_5_Pro,
        OpenRouterModel.Mistral_Large2,
        OpenRouterModel.Mixtral_8x7B,
        OpenRouterModel.Codestral,
        OpenRouterModel.GPT4_Turbo,
        OpenRouterModel.GPT4,
        OpenRouterModel.GPT3_5_Turbo,
        OpenRouterModel.o1_Preview,
        OpenRouterModel.o1_Mini,
        OpenRouterModel.Claude3_Opus,
        OpenRouterModel.Claude3_Sonnet,
        OpenRouterModel.Llama3_70B,
        OpenRouterModel.Llama3_8B,
        OpenRouterModel.Gemini_Pro,
        OpenRouterModel.Mistral_Nemo,
        OpenRouterModel.Mistral_7B,
        OpenRouterModel.Mixtral_8x22B,
        OpenRouterModel.Qwen2_72B,
        OpenRouterModel.DeepSeek_Coder,
        OpenRouterModel.CodeLlama_34B,
    ],
};

// Resources:
// - Claude Code CLI models - dynamically available through CLI
export enum ClaudeCodeModel {
    // Claude model aliases
    Sonnet = "sonnet",
    Opus = "opus",
    Haiku = "haiku",

    // Full model names
    Claude_Sonnet_4 = "claude-sonnet-4-20250514",
    Claude_3_5_Sonnet = "claude-3-5-sonnet-20241022",
    Claude_3_Opus = "claude-3-opus-20240229",
    Claude_3_Haiku = "claude-3-haiku-20240307",
}

export const claudeCodeServiceInfo: AIServiceInfo<ClaudeCodeModel> = {
    defaultModel: ClaudeCodeModel.Sonnet,
    enabled: false,
    name: "Claude Code",
    fallbackMaxTokens: 8_192,
    models: {
        [ClaudeCodeModel.Sonnet]: {
            enabled: true,
            name: "Claude_3_Sonnet_Name",
            descriptionShort: "Claude_3_Sonnet_Description_Short",
            inputCost: 0,    // Monthly subscription model - no per-token cost
            outputCost: 0,   // Monthly subscription model - no per-token cost
            contextWindow: 200_000,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
            },
            supportsReasoning: true,
        },
        [ClaudeCodeModel.Opus]: {
            enabled: true,
            name: "Claude_3_Opus_Name",
            descriptionShort: "Claude_3_Opus_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding capabilities" },
            },
            supportsReasoning: false,
        },
        [ClaudeCodeModel.Haiku]: {
            enabled: true,
            name: "Claude_3_Haiku_Name",
            descriptionShort: "Claude_3_Haiku_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
            },
            supportsReasoning: false,
        },
        [ClaudeCodeModel.Claude_Sonnet_4]: {
            enabled: true,
            name: "Claude_3_Sonnet_Name",
            descriptionShort: "Claude_3_Sonnet_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
            },
            supportsReasoning: true,
        },
        [ClaudeCodeModel.Claude_3_5_Sonnet]: {
            enabled: true,
            name: "Claude_3_5_Sonnet_Name",
            descriptionShort: "Claude_3_5_Sonnet_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 8_192,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
            },
            supportsReasoning: false,
        },
        [ClaudeCodeModel.Claude_3_Opus]: {
            enabled: true,
            name: "Claude_3_Opus_Name",
            descriptionShort: "Claude_3_Opus_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding capabilities" },
            },
            supportsReasoning: false,
        },
        [ClaudeCodeModel.Claude_3_Haiku]: {
            enabled: true,
            name: "Claude_3_Haiku_Name",
            descriptionShort: "Claude_3_Haiku_Description_Short",
            inputCost: 0,
            outputCost: 0,
            contextWindow: 200_000,
            maxOutputTokens: 4_096,
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution through Claude Code tools" },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access including Bash, Edit, Write, etc." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "File system access through built-in tools" },
            },
            supportsReasoning: false,
        },
    },
    displayOrder: [
        ClaudeCodeModel.Sonnet,
        ClaudeCodeModel.Claude_Sonnet_4,
        ClaudeCodeModel.Claude_3_5_Sonnet,
        ClaudeCodeModel.Opus,
        ClaudeCodeModel.Claude_3_Opus,
        ClaudeCodeModel.Haiku,
        ClaudeCodeModel.Claude_3_Haiku,
    ],
};

export const aiServicesInfo: AIServicesInfo = {
    defaultService: LlmServiceId.LocalOllama,
    services: {
        [AIServiceName.LocalOllama]: localOllamaServiceInfo,
        [AIServiceName.CloudflareGateway]: cloudflareGatewayServiceInfo,
        [AIServiceName.OpenRouter]: openRouterServiceInfo,
        [AIServiceName.ClaudeCode]: claudeCodeServiceInfo,
    },
    fallbacks: {
        // Universal fallback strategy: LocalOllama -> CloudflareGateway -> OpenRouter
        // LocalOllama models
        [LocalOllamaModel.Llama3_1_8B]: [CloudflareGatewayModel.Llama3_8B, OpenRouterModel.Llama3_1_8B],
        [LocalOllamaModel.Llama3_1_70B]: [CloudflareGatewayModel.Llama3_70B, OpenRouterModel.Llama3_1_70B],
        [LocalOllamaModel.CodeLlama_13B]: [CloudflareGatewayModel.GPT4o_Mini, OpenRouterModel.CodeLlama_34B],
        [LocalOllamaModel.Mistral_7B]: [CloudflareGatewayModel.Mistral_7B, OpenRouterModel.Mistral_7B],
        [LocalOllamaModel.Gemma2_9B]: [CloudflareGatewayModel.Gemma_7B, OpenRouterModel.Gemini_Pro],
        [LocalOllamaModel.Phi3_Medium]: [CloudflareGatewayModel.Phi2, OpenRouterModel.GPT4o_Mini],
        [LocalOllamaModel.Dynamic]: [CloudflareGatewayModel.GPT4o_Mini, OpenRouterModel.GPT4o_Mini],

        // CloudflareGateway models
        [CloudflareGatewayModel.GPT4o]: [OpenRouterModel.GPT4o],
        [CloudflareGatewayModel.GPT4o_Mini]: [OpenRouterModel.GPT4o_Mini],
        [CloudflareGatewayModel.GPT4_Turbo]: [OpenRouterModel.GPT4_Turbo],
        [CloudflareGatewayModel.GPT3_5_Turbo]: [OpenRouterModel.GPT3_5_Turbo],
        [CloudflareGatewayModel.Claude3_Sonnet]: [OpenRouterModel.Claude3_Sonnet],
        [CloudflareGatewayModel.Claude3_Haiku]: [OpenRouterModel.Claude3_Haiku],
        [CloudflareGatewayModel.Claude3_5_Sonnet]: [OpenRouterModel.Claude3_5_Sonnet],
        [CloudflareGatewayModel.Llama3_8B]: [OpenRouterModel.Llama3_8B],
        [CloudflareGatewayModel.Llama3_70B]: [OpenRouterModel.Llama3_70B],
        [CloudflareGatewayModel.Mistral_7B]: [OpenRouterModel.Mistral_7B],
        [CloudflareGatewayModel.Mistral_8x7B]: [OpenRouterModel.Mixtral_8x7B],
        [CloudflareGatewayModel.Phi2]: [OpenRouterModel.GPT4o_Mini],
        [CloudflareGatewayModel.Gemma_7B]: [OpenRouterModel.Gemini_Pro],

        // OpenRouter models - last in chain, no fallbacks
        [OpenRouterModel.GPT4o]: [],
        [OpenRouterModel.GPT4o_Mini]: [],
        [OpenRouterModel.GPT4_Turbo]: [],
        [OpenRouterModel.GPT4]: [],
        [OpenRouterModel.GPT3_5_Turbo]: [],
        [OpenRouterModel.o1_Preview]: [],
        [OpenRouterModel.o1_Mini]: [],
        [OpenRouterModel.Claude3_5_Sonnet]: [],
        [OpenRouterModel.Claude3_Opus]: [],
        [OpenRouterModel.Claude3_Sonnet]: [],
        [OpenRouterModel.Claude3_Haiku]: [],
        [OpenRouterModel.Gemini_1_5_Pro]: [],
        [OpenRouterModel.Gemini_1_5_Flash]: [],
        [OpenRouterModel.Gemini_Pro]: [],
        [OpenRouterModel.Llama3_1_405B]: [],
        [OpenRouterModel.Llama3_1_70B]: [],
        [OpenRouterModel.Llama3_1_8B]: [],
        [OpenRouterModel.Llama3_70B]: [],
        [OpenRouterModel.Llama3_8B]: [],
        [OpenRouterModel.Mistral_Large2]: [],
        [OpenRouterModel.Mistral_Nemo]: [],
        [OpenRouterModel.Mistral_7B]: [],
        [OpenRouterModel.Codestral]: [],
        [OpenRouterModel.Mixtral_8x7B]: [],
        [OpenRouterModel.Mixtral_8x22B]: [],
        [OpenRouterModel.Qwen2_72B]: [],
        [OpenRouterModel.DeepSeek_Coder]: [],
        [OpenRouterModel.CodeLlama_34B]: [],

        // ClaudeCode models - fallback to OpenRouter equivalents if available
        [ClaudeCodeModel.Sonnet]: [OpenRouterModel.Claude3_5_Sonnet],
        [ClaudeCodeModel.Opus]: [OpenRouterModel.Claude3_Opus],
        [ClaudeCodeModel.Haiku]: [OpenRouterModel.Claude3_Haiku],
        [ClaudeCodeModel.Claude_Sonnet_4]: [OpenRouterModel.Claude3_5_Sonnet],
        [ClaudeCodeModel.Claude_3_5_Sonnet]: [OpenRouterModel.Claude3_5_Sonnet],
        [ClaudeCodeModel.Claude_3_Opus]: [OpenRouterModel.Claude3_Opus],
        [ClaudeCodeModel.Claude_3_Haiku]: [OpenRouterModel.Claude3_Haiku],
    },
};
