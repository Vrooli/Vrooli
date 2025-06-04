import { type User } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { type TranslationFunc, type TranslationKeyService } from "../../types.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

const MIN_CREATIVITY = 0;
const MAX_CREATIVITY = 1;
export const DEFAULT_CREATIVITY = (MIN_CREATIVITY + MAX_CREATIVITY) / 2;

const MIN_VERBOSITY = 0;
const MAX_VERBOSITY = 1;
export const DEFAULT_VERBOSITY = (MIN_VERBOSITY + MAX_VERBOSITY) / 2;

const LATEST_CONFIG_VERSION = "1.0";

export const DEFAULT_PERSONA: Record<string, unknown> = {
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

// Defines the order of default persona fields in the UI and which ones are considered "default"
export const DEFAULT_PERSONA_UI_ORDER = [
    "occupation",
    "persona",
    "tone",
    "startingMessage",
    "keyPhrases",
    "domainKnowledge",
    "bias",
    // creativity and verbosity are handled by sliders, not direct text inputs in persona object in UI
];

export interface BotConfigObject extends BaseConfigObject {
    model?: string;
    maxTokens?: number;
    persona?: Record<string, unknown>;
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
        bot: Pick<User, "botSettings"> | null | undefined,
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): BotConfig {
        const botSettings = bot?.botSettings;
        return super.parseBase<BotConfigObject, BotConfig>(
            botSettings,
            logger,
            (cfg) => {
                if (opts?.useFallbacks ?? true) {
                    // If persona exists, merge it with defaults, otherwise use defaults
                    cfg.persona = { ...DEFAULT_PERSONA, ...(cfg.persona ?? {}) };
                } else if (cfg.persona === undefined) {
                    // If fallbacks are off and persona is undefined, ensure it's not null
                    cfg.persona = undefined;
                }
                return new BotConfig({ botSettings: cfg });
            },
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

export function getModelName(option: LlmModel | null, t: TranslationFunc) {
    return option ? t(option.name, { ns: "service" }) : "";
}
export function getModelDescription(option: LlmModel, t: TranslationFunc) {
    return option && option.description ? t(option.description, { ns: "service" }) : "";
}
