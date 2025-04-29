import { BotSettingsConfig, BotSettingsTranslation, DeleteType, generatePKString, generatePublicId } from "@local/shared";
import { noEmptyString, toBool, validNumber } from "../../../builders/noNull.js";
import { logger } from "../../../events/logger.js";
import { LlmTaskConverters } from "../converter.js";

export const convert: LlmTaskConverters = {
    BotAdd: (data) => ({
        id: generatePKString(),
        publicId: generatePublicId(),
        isBotDepictingPerson: false,
        isPrivate: true,
        name: noEmptyString(data.name) ?? "Bot",
        translationsCreate: noEmptyString(data.bio) ? [{
            id: generatePKString(),
            language: "en",
            bio: noEmptyString(data.bio),
        }] : undefined,
        botSettings: JSON.stringify({
            creativity: validNumber(data.creativity, 0),
            verbosity: validNumber(data.verbosity, 0),
            translations: {
                en: {
                    occupation: noEmptyString(data.occupation),
                    persona: noEmptyString(data.persona),
                    startingMessage: noEmptyString(data.startingMessage),
                    tone: noEmptyString(data.tone),
                    keyPhrases: noEmptyString(data.keyPhrases),
                    domainKnowledge: noEmptyString(data.domainKnowledge),
                    bias: noEmptyString(data.bias),
                },
            },

        }),
    }),
    BotDelete: (data) => ({
        id: noEmptyString(data.id) ?? "",
        objectType: DeleteType.User,
    }),
    BotFind: (data) => ({
        searchString: noEmptyString(data.searchString),
        memberInTeamId: noEmptyString(data.memberInTeamId),
    }),
    BotUpdate: (data, language, existing) => {
        const botSettings = BotSettingsConfig.deserialize(existing, logger);
        const updatedBotSettingsSchema = {
            ...botSettings.schema,
            model: noEmptyString(data.model ?? botSettings.schema.model),
            maxTokens: validNumber(data.maxTokens ?? botSettings.schema.maxTokens),
            creativity: BotSettingsConfig.parseCreativity(data.creativity ?? botSettings.schema.creativity),
            verbosity: BotSettingsConfig.parseVerbosity(data.verbosity ?? botSettings.schema.verbosity),
            translations: botSettings.schema.translations ?? {},
        };
        const updatedTranslations = (updatedBotSettingsSchema.translations ?? {}) as Record<string, BotSettingsTranslation>;
        if (!updatedTranslations[language]) {
            updatedTranslations[language] = {};
        }
        if (data.bias) {
            updatedTranslations[language].bias = noEmptyString(data.bias);
        }
        if (data.creativity) {
            updatedTranslations[language].creativity = BotSettingsConfig.parseCreativity(data.creativity);
        }
        if (data.verbosity) {
            updatedTranslations[language].verbosity = BotSettingsConfig.parseVerbosity(data.verbosity);
        }
        if (data.domainKnowledge) {
            updatedTranslations[language].domainKnowledge = noEmptyString(data.domainKnowledge);
        }
        if (data.keyPhrases) {
            updatedTranslations[language].keyPhrases = noEmptyString(data.keyPhrases);
        }
        if (data.occupation) {
            updatedTranslations[language].occupation = noEmptyString(data.occupation);
        }
        if (data.persona) {
            updatedTranslations[language].persona = noEmptyString(data.persona);
        }
        if (data.startingMessage) {
            updatedTranslations[language].startingMessage = noEmptyString(data.startingMessage);
        }
        if (data.tone) {
            updatedTranslations[language].tone = noEmptyString(data.tone);
        }
        updatedBotSettingsSchema.translations = updatedTranslations;
        botSettings.schema = updatedBotSettingsSchema;

        return {
            id: data.id + "",
            isBotDepictingPerson: toBool(data.isBotDepictingPerson ?? existing.isBotDepictingPerson),
            name: noEmptyString(data.name),
            translationsUpdate: noEmptyString(data.bio) ? [{
                id: generatePKString(),
                language: "en",
                ...existing.translations?.find(t => t.language === "en"),
                bio: noEmptyString(data.bio),
            }] : undefined,
            botSettings: botSettings.serialize("json"),
        };
    },
    MembersAdd: (data) => ({
        id: generatePKString(),
        publicId: generatePublicId(),
        //...
    }),
    MembersDelete: (data) => {
        let ids = Array.isArray(data.ids) ?
            data.ids :
            typeof data.ids === "string" ?
                [data.ids] :
                [];
        ids = ids.filter(uuidValidate);
        const objects = ids.map(id => ({
            id,
            objectType: DeleteType.Member,
        }));
        return { objects };
    },
    MembersFind: (data) => ({
        searchString: noEmptyString(data.searchString),
        teamId: noEmptyString(data.teamId),
    }),
    MembersUpdate: (data) => ({
        id: noEmptyString(data.id) ?? "",
        //...
    }),
    ReminderAdd: (data) => ({
        id: generatePKString(),
        name: noEmptyString(data.name) ?? "Reminder",
        description: noEmptyString(data.description),
        index: -1, // TODO
        // TODO need list ID from user session data
        //...
    }),
    ReminderDelete: (data) => ({
        id: noEmptyString(data.id) ?? "",
        objectType: DeleteType.Reminder,
    }),
    ReminderFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ReminderUpdate: (data) => ({
        id: noEmptyString(data.id) ?? "",
        //...
    }),
    ResourceAdd: (data) => ({
        id: generatePKString(),
        publicId: generatePublicId(),
        isPrivate: true,
        //...
    }),
    ResourceDelete: (data) => ({
        id: noEmptyString(data.id) ?? "",
        objectType: DeleteType.Resource,
    }),
    ResourceFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ResourceUpdate: (data) => ({
        id: noEmptyString(data.id) ?? "",
        isPrivate: true,
        //...
    }),
    RunStart: (data) => ({
        //...
    } as any),
    ScheduleAdd: (data) => ({
        id: generatePKString(),
        publicId: generatePublicId(),
        timezone: "", //TODO need timezone from user session
        //...
    }),
    ScheduleDelete: (data) => ({
        id: noEmptyString(data.id) ?? "",
        objectType: DeleteType.Schedule,
    }),
    ScheduleFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ScheduleUpdate: (data) => ({
        id: noEmptyString(data.id) ?? "",
        //...
    }),
    Start: (data) => ({}),
    TeamAdd: (data) => ({
        id: generatePKString(),
        publicId: generatePublicId(),
        isPrivate: true,
        //...
    }),
    TeamDelete: (data) => ({
        id: noEmptyString(data.id) ?? "",
        objectType: DeleteType.Team,
    }),
    TeamFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    TeamUpdate: (data) => ({
        id: noEmptyString(data.id) ?? "",
        //...
    }),
};
