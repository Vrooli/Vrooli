import { type TranslationKeyService } from "../types.js";

export enum AIServiceName {
    OpenAI = "OpenAI",
    Anthropic = "Anthropic",
    Mistral = "Mistral",
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

export type LlmServiceModel = AnthropicModel | MistralModel | OpenAIModel;

/**
 * All available services
 */
export enum LlmServiceId {
    Anthropic = "Anthropic",
    Mistral = "Mistral",
    OpenAI = "OpenAI",
}

/**
 * Information about all available AI services
 */
export type AIServicesInfo = {
    defaultService: LlmServiceId;
    services: {
        [AIServiceName.OpenAI]: AIServiceInfo<OpenAIModel>,
        [AIServiceName.Anthropic]: AIServiceInfo<AnthropicModel>,
        [AIServiceName.Mistral]: AIServiceInfo<MistralModel>,
    },
    fallbacks: Record<LlmServiceModel, LlmServiceModel[]>,
};

// Resources: 
// - https://platform.openai.com/docs/models
// - https://openai.com/api/pricing/https://openai.com/api/pricing/
export enum OpenAIModel {
    Gpt4o_Mini = "gpt-4o-mini-2024-07-18",
    Gpt4o = "gpt-4o-2024-05-13",
    Gpt4 = "gpt-4-0125-preview",
    Gpt4_Turbo = "gpt-4-turbo-2024-04-09",
    o1_Mini = "o1-mini-2024-09-12",
    o1_Preview = "o1-preview-2024-09-12",
    // DallE3 = "dall-e-3",
    // Whisper = "whisper-1",
    // Gpt3_5Turbo = "gpt-3.5-turbo-0125", // Deprecated
}
export const openAIServiceInfo: AIServiceInfo<OpenAIModel> = {
    defaultModel: OpenAIModel.Gpt4o_Mini,
    enabled: true,
    name: "OpenAI",
    fallbackMaxTokens: 4_096, // 4K tokens
    models: {
        [OpenAIModel.Gpt4o_Mini]: {
            enabled: true,
            name: "GPT_4o_Mini_Name",
            descriptionShort: "GPT_4o_Mini_Description_Short",
            inputCost: 150,          // $0.15
            outputCost: 60,          // $0.60
            contextWindow: 128_000,  // 128K tokens
            maxOutputTokens: 16_384, // 16K tokens
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: false,
        },
        [OpenAIModel.Gpt4o]: {
            enabled: true,
            name: "GPT_4o_Name",
            descriptionShort: "GPT_4o_Description_Short",
            inputCost: 250,          // $2.50 per 1M tokens (was 500)
            outputCost: 1_000,       // $10.00 per 1M tokens (was 1500)
            contextWindow: 128_000,  // 128K tokens
            maxOutputTokens: 4_096,  // 4K tokens
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: false,
        },
        [OpenAIModel.Gpt4]: {
            enabled: true,
            name: "GPT_4_Name",
            descriptionShort: "GPT_4_Description_Short",
            inputCost: 3_000,        // $30.00
            outputCost: 6_000,       // $60.00
            contextWindow: 8_192,    // 8K tokens
            maxOutputTokens: 8_192,  // 8K tokens
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: false,
        },
        [OpenAIModel.Gpt4_Turbo]: {
            enabled: true,
            name: "GPT_4_Turbo_Name",
            descriptionShort: "GPT_4_Turbo_Description_Short",
            inputCost: 1_000,        // $10.00
            outputCost: 3_000,       // $30.00
            contextWindow: 128_000,  // 128K tokens
            maxOutputTokens: 4_096,  // 4K tokens
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: false,
        },
        [OpenAIModel.o1_Mini]: {
            enabled: true,
            name: "o1_Mini_Name",
            descriptionShort: "o1_Mini_Description_Short",
            inputCost: 300,          // $3.00
            outputCost: 1_200,       // $12.00   
            contextWindow: 128_000,  // 128K tokens
            maxOutputTokens: 65_536, // 65K tokens 
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: true,
        },
        [OpenAIModel.o1_Preview]: {
            enabled: true,
            name: "o1_Preview_Name",
            descriptionShort: "o1_Preview_Description_Short",
            inputCost: 1_500,        // $15.00
            outputCost: 6_000,       // $60.00
            contextWindow: 128_000,  // 128K tokens
            maxOutputTokens: 32_768, // 32K tokens
            features: {
                [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.FileSearch]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing, potentially with storage fees for persistent capabilities." },
                [ModelFeature.FunctionCalling]: { type: "generic", notes: "Enabled. Cost is part of the model's standard token pricing." },
                [ModelFeature.Vision]: { type: "vision", notes: "Image understanding is priced based on token consumption as per OpenAI's image tokenization rules." },
            },
            supportsReasoning: true,
        },
        // [OpenAIModel.DallE3]: {
        //     enabled: true,
        //     name: "DALL_E_3_Name" as any,
        //     descriptionShort: "DALL_E_3_Description_Short" as any,
        //     inputCost: 0,
        //     outputCost: 0,
        //     contextWindow: 0,
        //     maxOutputTokens: 0,
        //     features: {
        //         [ModelFeature.ImageGeneration]: {
        //             type: "image_generation",
        //             modelName: "DALL-E 3",
        //             costs: [
        //                 { quality: "standard", resolution: "1024x1024", costPerImage: 4 },
        //                 { quality: "standard", resolution: "1024x1792", costPerImage: 8 },
        //                 { quality: "hd", resolution: "1024x1024", costPerImage: 8 },
        //                 { quality: "hd", resolution: "1024x1792", costPerImage: 12 },
        //             ]
        //         }
        //     }
        // },
        // [OpenAIModel.Whisper]: {
        //     enabled: true,
        //     name: "Whisper_Name" as any,
        //     descriptionShort: "Whisper_Description_Short" as any,
        //     inputCost: 0,
        //     outputCost: 0,
        //     contextWindow: 0,
        //     maxOutputTokens: 0,
        //     features: {
        //         [ModelFeature.Transcription]: {
        //             type: "transcription",
        //             modelName: "Whisper",
        //             costPerMinute: 0.6
        //         }
        //     }
        // }
    },
    displayOrder: [
        OpenAIModel.Gpt4o_Mini,
        OpenAIModel.Gpt4o,
        OpenAIModel.Gpt4_Turbo,
        OpenAIModel.Gpt4,
        OpenAIModel.o1_Mini,
        OpenAIModel.o1_Preview,
        // OpenAIModel.DallE3,
        // OpenAIModel.Whisper,
    ],
};

// Resources: 
// - https://docs.anthropic.com/en/docs/about-claude/models#model-comparison-table
export enum AnthropicModel {
    Haiku3 = "claude-3-haiku-20240307",
    Opus3 = "claude-3-opus-20240229",
    Sonnet3 = "claude-3-sonnet-20240229",
    Sonnet3_5 = "claude-3-5-sonnet-20240620",
}
export const anthropicServiceInfo: AIServiceInfo<AnthropicModel> = {
    defaultModel: AnthropicModel.Sonnet3_5,
    enabled: true,
    name: "Anthropic",
    fallbackMaxTokens: 4_096, // 4K tokens
    models: {
        [AnthropicModel.Haiku3]: {
            enabled: true,
            name: "Claude_3_Haiku_Name",
            descriptionShort: "Claude_3_Haiku_Description_Short",
            inputCost: 25,          // $0.25
            outputCost: 125,        // $1.25
            contextWindow: 200_000, // 200K tokens
            maxOutputTokens: 4_096, // 4K tokens
            features: {
                [ModelFeature.Vision]: { type: "vision", notes: "Vision capabilities are supported; pricing is typically part of token consumption." },
            },
            supportsReasoning: false,
        },
        [AnthropicModel.Opus3]: {
            enabled: true,
            name: "Claude_3_Opus_Name",
            descriptionShort: "Claude_3_Opus_Description_Short",
            inputCost: 1_500,       // $15.00
            outputCost: 7_500,      // $75.00
            contextWindow: 200_000, // 200K tokens
            maxOutputTokens: 4_096, // 4K tokens
            features: {
                [ModelFeature.Vision]: { type: "vision", notes: "Vision capabilities are supported; pricing is typically part of token consumption." },
            },
            supportsReasoning: false,
        },
        [AnthropicModel.Sonnet3]: {
            enabled: true,
            name: "Claude_3_Sonnet_Name",
            descriptionShort: "Claude_3_Sonnet_Description_Short",
            inputCost: 300,         // $3.00
            outputCost: 1_500,      // $15.00
            contextWindow: 200_000, // 200K tokens
            maxOutputTokens: 4_096, // 4K tokens
            features: {
                [ModelFeature.Vision]: { type: "vision", notes: "Vision capabilities are supported; pricing is typically part of token consumption." },
            },
            supportsReasoning: false,
        },
        [AnthropicModel.Sonnet3_5]: {
            enabled: true,
            name: "Claude_3_5_Sonnet_Name",
            descriptionShort: "Claude_3_5_Sonnet_Description_Short",
            inputCost: 300,         // $3.00
            outputCost: 1_500,      // $15.00
            contextWindow: 200_000, // 200K tokens
            maxOutputTokens: 8_192, // 8K tokens
            features: {
                [ModelFeature.Vision]: { type: "vision", notes: "Vision capabilities are supported; pricing is typically part of token consumption." },
            },
            supportsReasoning: false,
        },
    },
    displayOrder: [
        AnthropicModel.Haiku3,
        AnthropicModel.Sonnet3_5,
        AnthropicModel.Opus3,
        AnthropicModel.Sonnet3,
    ],
};

// Resources:
// - https://mistral.ai/technology/#pricing
export enum MistralModel {
    // _8x7b = "open-mixtral-8x7b", // Deprecated
    // _7b = "open-mistral-7b", // Deprecated
    Codestral = "codestral-2405",
    Large2 = "mistral-large-2407",
    Nemo = "open-mistral-nemo-2407",
}
export const mistralServiceInfo: AIServiceInfo<MistralModel> = {
    defaultModel: MistralModel.Nemo,
    enabled: true,
    name: "Mistral",
    fallbackMaxTokens: 4_096, // 4K tokens
    models: {
        [MistralModel.Codestral]: {
            enabled: true,
            name: "Mistral_Codestral_Name",
            descriptionShort: "Mistral_Codestral_Description_Short",
            inputCost: 20,         // $0.20
            outputCost: 60,        // $0.60
            contextWindow: 32_000,  // 32K tokens
            maxOutputTokens: 4_096, // NOTE: Couldn't find the actual value
            features: {},
            supportsReasoning: false,
        },
        [MistralModel.Large2]: {
            enabled: true,
            name: "Mistral_Large_2_Name",
            descriptionShort: "Mistral_Large_2_Description_Short",
            inputCost: 200,         // $2.00
            outputCost: 600,        // $6.00
            contextWindow: 128_000, // 128K tokens
            maxOutputTokens: 4_096, // NOTE: Couldn't find the actual value
            features: {},
            supportsReasoning: false,
        },
        [MistralModel.Nemo]: {
            enabled: true,
            name: "Mistral_Nemo_Name",
            descriptionShort: "Mistral_Nemo_Description_Short",
            inputCost: 15,          // $0.15
            outputCost: 15,         // $0.15
            contextWindow: 128_000, // 128K tokens
            maxOutputTokens: 4_096, // NOTE: Couldn't find the actual value
            features: {},
            supportsReasoning: false,
        },
    },
    displayOrder: [
        MistralModel.Nemo,
        MistralModel.Codestral,
        MistralModel.Large2,
    ],
};

export const aiServicesInfo: AIServicesInfo = {
    defaultService: LlmServiceId.OpenAI,
    services: {
        [AIServiceName.OpenAI]: openAIServiceInfo,
        [AIServiceName.Anthropic]: anthropicServiceInfo,
        [AIServiceName.Mistral]: mistralServiceInfo,
    },
    fallbacks: {
        [AnthropicModel.Haiku3]: [OpenAIModel.Gpt4o_Mini, MistralModel.Nemo],
        [AnthropicModel.Opus3]: [OpenAIModel.Gpt4_Turbo, MistralModel.Large2],
        [AnthropicModel.Sonnet3]: [OpenAIModel.Gpt4o, MistralModel.Nemo],
        [AnthropicModel.Sonnet3_5]: [OpenAIModel.Gpt4o, MistralModel.Nemo],
        [MistralModel.Codestral]: [OpenAIModel.Gpt4o, AnthropicModel.Sonnet3_5],
        [MistralModel.Large2]: [OpenAIModel.Gpt4_Turbo, AnthropicModel.Opus3],
        [MistralModel.Nemo]: [OpenAIModel.Gpt4o_Mini, AnthropicModel.Haiku3],
        [OpenAIModel.Gpt4o_Mini]: [AnthropicModel.Haiku3, MistralModel.Nemo],
        [OpenAIModel.Gpt4o]: [AnthropicModel.Sonnet3_5, MistralModel.Nemo],
        [OpenAIModel.Gpt4]: [AnthropicModel.Opus3, MistralModel.Large2],
        [OpenAIModel.Gpt4_Turbo]: [AnthropicModel.Opus3, MistralModel.Large2],
        [OpenAIModel.o1_Mini]: [AnthropicModel.Haiku3, MistralModel.Nemo],
        [OpenAIModel.o1_Preview]: [AnthropicModel.Sonnet3_5, MistralModel.Large2],
    },
};
