import { type AIServiceName, type AIServicesInfo } from "../../ai/services.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { type TranslationFunc, type TranslationKeyService } from "../../types.js";
import { BaseConfig, type BaseConfigObject } from "./baseConfig.js";
import { type StringifyMode } from "./utils.js";

const MIN_CREATIVITY = 0;
const MAX_CREATIVITY = 1;
export const DEFAULT_CREATIVITY = (MIN_CREATIVITY + MAX_CREATIVITY) / 2;

const MIN_VERBOSITY = 0;
const MAX_VERBOSITY = 1;
export const DEFAULT_VERBOSITY = (MIN_VERBOSITY + MAX_VERBOSITY) / 2;

const LATEST_CONFIG_VERSION = "1.0";

const DEFAULT_PERSONA: Record<string, unknown> = {
    bias: "",
    creativity: 0,
    domainKnowledge: "",
    keyPhrases: "",
    occupation: "",
    persona: "",
    startingMessage: "",
    tone: "",
    verbosity: 0,
};

export interface BotConfigObject extends BaseConfigObject {
    model?: string;
    maxTokens?: number;
    persona: Record<string, unknown>;
}

export class BotConfig extends BaseConfig<BotConfigObject> {
    model?: BotConfigObject["model"];
    maxTokens?: BotConfigObject["maxTokens"];
    persona: BotConfigObject["persona"];

    constructor({ botSettings }: { botSettings: BotConfigObject }) {
        super(botSettings);
        this.model = botSettings.model;
        this.maxTokens = botSettings.maxTokens;
        this.persona = botSettings.persona;
    }

    static parse(
        bot: { botSettings: BotConfigObject },
        logger: PassableLogger,
        opts?: { mode?: StringifyMode; useFallbacks?: boolean },
    ): BotConfig {
        return super.parseBase<BotConfigObject, BotConfig>(
            bot.botSettings,
            logger,
            (cfg) => {
                if (opts?.useFallbacks ?? true) {
                    cfg.persona ??= { ...DEFAULT_PERSONA };
                }
                return new BotConfig({ botSettings: cfg });
            },
            { mode: opts?.mode },
        );
    }

    static default(): BotConfig {
        return new BotConfig({
            botSettings: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                model: undefined,
                maxTokens: undefined,
                persona: { ...DEFAULT_PERSONA },
            },
        });
    }

    export(): BotConfigObject {
        return {
            ...super.export(),
            model: this.model,
            maxTokens: this.maxTokens,
            persona: this.persona,
        };
    }
}

export type LlmModel = {
    name: TranslationKeyService,
    description?: TranslationKeyService,
    value: string,
};

export function getAvailableModels(aiServicesInfo: AIServicesInfo | null | undefined): LlmModel[] {
    const models: LlmModel[] = [];
    if (!aiServicesInfo) return models;
    const services = aiServicesInfo.services;
    for (const serviceKey in services) {
        const service = services[serviceKey as AIServiceName];
        if (service.enabled) {
            for (const modelKey of service.displayOrder) {
                const modelInfo = service.models[modelKey];
                if (modelInfo.enabled) {
                    models.push({
                        name: modelInfo.name, // This is a ServiceKey for i18next
                        description: modelInfo.descriptionShort, // Also a ServiceKey
                        value: modelKey,
                    });
                }
            }
        }
    }
    return models;
}

export function getModelName(option: LlmModel | null, t: TranslationFunc) {
    return option ? t(option.name, { ns: "service" }) : "";
}
export function getModelDescription(option: LlmModel, t: TranslationFunc) {
    return option && option.description ? t(option.description, { ns: "service" }) : "";
}
