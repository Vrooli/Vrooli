import { aiServicesInfo, type AIServiceName, type AIServicesInfo } from "../../ai/services.js";
import { User, UserTranslation } from "../../api/types.js";
import { PassableLogger } from "../../consts/commonTypes.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import { type TranslationFunc, type TranslationKeyService } from "../../types.js";
import { toDouble } from "../../validation/utils/builders/convert.js";
import { type BotShape } from "../models/models.js";
import { parseObject, stringifyObject, type StringifyMode } from "./utils.js";

const MIN_CREATIVITY = 0;
const MAX_CREATIVITY = 1;
export const DEFAULT_CREATIVITY = (MIN_CREATIVITY + MAX_CREATIVITY) / 2;

const MIN_VERBOSITY = 0;
const MAX_VERBOSITY = 1;
export const DEFAULT_VERBOSITY = (MIN_VERBOSITY + MAX_VERBOSITY) / 2;

/**
 * The translation object for a bot's settings.
 * 
 * This is a good set of parameters to specify a bot's personality, 
 * and is understandable by an AI and users. BUT, we need to make 
 * sure that we support any arbitrary parameters that someone might 
 * want to store.
 */
export type BotSettingsTranslation = {
    bias?: string;
    creativity?: number;
    domainKnowledge?: string;
    keyPhrases?: string;
    occupation?: string;
    persona?: string;
    startingMessage?: string;
    tone?: string;
    verbosity?: number;
};

/**
 * Deserialized bot settings, ready to use for AI response generation.
 */
export type BotSettings = {
    /** The bot's preferred model */
    model?: string;
    /** 
     * The bot's custom maxTokens value.
     * 
     * It will be Math.min'd with the model's maxTokens value.
     */
    maxTokens?: number;
    /** 
     * The bot's display name.
     * 
     * Can influence generation style if the name is well-known (e.g. "Shakespeare").
     */
    name: string;
    /**
     * Creativity value for the bot, from `MIN_CREATIVITY` to `MAX_CREATIVITY`.
     * 
     * Can be overridden by a translation value.
     */
    creativity?: number;
    /**
     * Verbosity value for the bot, from `MIN_VERBOSITY` to `MAX_VERBOSITY`.
     * 
     * Can be overridden by a translation value.
     */
    verbosity?: number;
    /**
     * Translations for the bot's settings.
     * 
     * Only one translation should be provided to the AI, chosen using 
     * the user's preferred languages.
     */
    translations?: Record<string, BotSettingsTranslation | object>;
};

export interface BotSettingsConfigObject {
    __version: string;
    schema: BotSettings;
}

/**
 * Configuration class for serializing and deserializing bot settings.
 */
export class BotSettingsConfig {
    static readonly LATEST_VERSION = "1.0";

    __version: string;
    schema: BotSettings;

    constructor(data: BotSettingsConfigObject) {
        this.__version = data.__version ?? BotSettingsConfig.LATEST_VERSION;
        this.schema = data.schema;
    }

    /**
     * Deserialize a stored string into a BotSettingsConfig.
     * 
     * NOTE: Unlike other config deserializers, this one adds additional data from the User object 
     * such as translations, name, etc. This means that the schema returned is slightly different than 
     * the schema used in the serializer
     */
    static deserialize(
        botData: { botSettings?: string | null | undefined, name?: string | null | undefined, translations?: Partial<UserTranslation>[] | null | undefined } | null | undefined,
        logger: PassableLogger,
        { mode = "json" }: { mode?: StringifyMode } = {},
    ): BotSettingsConfig {
        let obj = botData?.botSettings
            ? parseObject<BotSettingsConfigObject>(botData.botSettings, mode, logger)
            : null;
        // Set defaults if the object is ill-formed
        if (
            !obj ||
            typeof obj !== "object" ||
            !Object.prototype.hasOwnProperty.call(obj, "__version") ||
            typeof obj.__version !== "string" ||
            !Object.prototype.hasOwnProperty.call(obj, "schema") ||
            typeof obj.schema !== "object"
        ) {
            obj = {
                __version: BotSettingsConfig.LATEST_VERSION,
                schema: BotSettingsConfig.defaultBotSettings(),
            };
        }
        if (botData?.name) {
            obj.schema.name = botData.name;
        }
        if (!obj.schema.name) {
            obj.schema.name = "";
        }
        // The User translations may contain botSettings translation information that we need to merge in.
        if (botData?.translations && Array.isArray(botData.translations) && botData.translations.length > 0) {
            for (const translation of botData.translations) {
                if (!translation.language) {
                    continue;
                }
                // Exclude extraneous fields like __typename, id, bio, etc.
                const details = Object.assign({}, translation);
                delete details.__typename;
                delete details.id;
                delete details.language;
                delete details.bio;

                if (Object.keys(details).length > 0) {
                    if (!obj.schema.translations) {
                        obj.schema.translations = {};
                    }
                    obj.schema.translations[translation.language] = {
                        ...obj.schema.translations[translation.language],
                        ...details,
                    };
                }
            }
        }
        // Make sure creativity/verbosity are numbers, both at the top level and in translations.
        if (Object.prototype.hasOwnProperty.call(obj.schema, "creativity")) {
            obj.schema.creativity = BotSettingsConfig.parseCreativity(obj.schema.creativity);
        }
        if (Object.prototype.hasOwnProperty.call(obj.schema, "verbosity")) {
            obj.schema.verbosity = BotSettingsConfig.parseVerbosity(obj.schema.verbosity);
        }
        for (const translation of Object.values(obj.schema.translations ?? {})) {
            if (typeof translation === "object") {
                if (Object.prototype.hasOwnProperty.call(translation, "creativity")) {
                    (translation as { creativity?: number }).creativity = BotSettingsConfig.parseCreativity((translation as { creativity?: string | number }).creativity);
                }
                if (Object.prototype.hasOwnProperty.call(translation, "verbosity")) {
                    (translation as { verbosity?: number }).verbosity = BotSettingsConfig.parseVerbosity((translation as { verbosity?: string | number }).verbosity);
                }
            }
        }
        return new BotSettingsConfig(obj);
    }

    /**
     * Serialize the configuration back into a string.
     */
    serialize(mode: StringifyMode = "json"): string {
        // Get the export object
        const obj = this.export() as { schema: { name?: string } };
        // Exclude the name from the object schema
        delete obj.schema.name;
        // Serialize the object
        return stringifyObject(obj, mode);
    }

    export(): BotSettingsConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }

    /**
     * Parses a `creativity` value from a string or number, 
     * ensuring it's between `MIN_CREATIVITY` and `MAX_CREATIVITY`.
     * 
     * @param value The value to parse.
     * @returns The parsed value, or `DEFAULT_CREATIVITY` if it's not a number or is out of range.
     */
    static parseCreativity(value: unknown): number {
        if (typeof value !== "number" && typeof value !== "string") {
            return DEFAULT_CREATIVITY;
        }
        const providedCreativity = toDouble(value);
        // Make sure it's in the correct range
        return Math.min(Math.max(providedCreativity, MIN_CREATIVITY), MAX_CREATIVITY);
    }

    /**
     * Parses a `verbosity` value from a string or number, 
     * ensuring it's between `MIN_VERBOSITY` and `MAX_VERBOSITY`.
     */
    static parseVerbosity(value: unknown): number {
        if (typeof value !== "number" && typeof value !== "string") {
            return DEFAULT_VERBOSITY;
        }
        const providedVerbosity = toDouble(value);
        // Make sure it's in the correct range
        return Math.min(Math.max(providedVerbosity, MIN_VERBOSITY), MAX_VERBOSITY);
    }

    static defaultBotSettings(): BotSettings {
        return {
            name: "",
            model: undefined,
            translations: {},
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

/**
 * Creates a normalized view of the bot settings.
 * 
 * This is useful for the bot form, as formik requires fields to be present.
 * 
 * @param language The language to use for the translations.
 * @param availableModels The available models.
 * @param existing The existing bot data.
 * @returns The normalized bot data.
 */
export function findBotDataForForm(
    language: string,
    availableModels: LlmModel[],
    existing?: Partial<User> | BotShape | null | undefined,
): Required<Omit<BotSettings, "maxTokens" | "name" | "translations">> & { translations: Array<BotSettingsTranslation & { language: string; __typename: string; id: string; bio?: string }> } {
    const schema = existing ?
        BotSettingsConfig.deserialize(existing, console).export().schema
        : BotSettingsConfig.defaultBotSettings();
    const settingsTranslation = schema.translations?.[language] as Partial<BotSettingsTranslation> | undefined;
    const creativity = BotSettingsConfig.parseCreativity(settingsTranslation?.creativity ?? schema.creativity);
    const verbosity = BotSettingsConfig.parseVerbosity(settingsTranslation?.verbosity ?? schema.verbosity);

    const defaultTranslation = {
        bio: "",
        bias: settingsTranslation?.bias ?? "",
        domainKnowledge: settingsTranslation?.domainKnowledge ?? "",
        keyPhrases: settingsTranslation?.keyPhrases ?? "",
        occupation: settingsTranslation?.occupation ?? "",
        persona: settingsTranslation?.persona ?? "",
        startingMessage: settingsTranslation?.startingMessage ?? "",
        tone: settingsTranslation?.tone ?? "",
        __typename: "UserTranslation" as const,
        id: DUMMY_ID,
        language,
    };

    const translations: Array<BotSettingsTranslation & { language: string; __typename: string; id: string; bio?: string }> = [];
    if (schema.translations) {
        Object.entries(schema.translations).forEach(([lang, botTranslation]) => {
            translations.push({
                ...defaultTranslation,
                ...botTranslation,
                language: lang,
            });
        });
    }
    // Ensure the requested language exists.
    if (!translations.some((t) => t.language === language)) {
        translations.push(defaultTranslation);
    }
    const model =
        typeof schema.model === "string" &&
            availableModels.some((m) => m.value === schema.model)
            ? schema.model
            : (aiServicesInfo.services[aiServicesInfo.defaultService]?.defaultModel ?? "");
    return {
        creativity,
        verbosity,
        model,
        translations,
    };
}
