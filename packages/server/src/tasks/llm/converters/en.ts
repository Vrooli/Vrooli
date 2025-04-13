import { BotSettingsConfig, BotSettingsTranslation, DeleteType, uuid, uuidValidate } from "@local/shared";
import { noEmptyString, toBool, validNumber, validUuid } from "../../../builders/noNull.js";
import { logger } from "../../../events/logger.js";
import { LlmTaskConverters } from "../converter.js";

export const convert: LlmTaskConverters = {
    ApiAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    ApiDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Api,
    }),
    ApiFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ApiUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    BotAdd: (data) => ({
        id: uuid(),
        isBotDepictingPerson: false,
        isPrivate: true,
        name: noEmptyString(data.name) ?? "Bot",
        translationsCreate: noEmptyString(data.bio) ? [{
            id: uuid(),
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
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.User,
    }),
    BotFind: (data) => ({
        searchString: noEmptyString(data.searchString),
        memberInTeamId: validUuid(data.memberInTeamId),
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
                id: uuid(),
                language: "en",
                ...existing.translations?.find(t => t.language === "en"),
                bio: noEmptyString(data.bio),
            }] : undefined,
            botSettings: botSettings.serialize("json"),
        };
    },
    DataConverterAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    DataConverterDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Code,
    }),
    DataConverterFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    DataConverterUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    MembersAdd: (data) => ({
        id: uuid(),
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
        teamId: validUuid(data.teamId),
    }),
    MembersUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    NoteAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    NoteDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Note,
    }),
    NoteFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    NoteUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    ProjectAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    ProjectDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Project,
    }),
    ProjectFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ProjectUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    QuestionAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    QuestionDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Question,
    }),
    QuestionFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    QuestionUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    ReminderAdd: (data) => ({
        id: uuid(),
        name: noEmptyString(data.name) ?? "Reminder",
        description: noEmptyString(data.description),
        index: -1, // TODO
        // TODO need list ID from user session data
        //...
    }),
    ReminderDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Reminder,
    }),
    ReminderFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ReminderUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    RoleAdd: (data, language) => {
        const description = noEmptyString(data.description);
        return {
            id: uuid(),
            name: noEmptyString(data.name) ?? "Role",
            teamConnect: validUuid(data.teamId) ?? "",
            permissions: JSON.stringify({}),
            translationsCreate: description ? [{
                id: uuid(),
                language,
                description,
            }] : [],
            //...
        };
    },
    RoleDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Role,
    }),
    RoleFind: (data) => ({
        searchString: noEmptyString(data.searchString),
        teamId: validUuid(data.teamId) ?? "",
    }),
    RoleUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    RoutineAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    RoutineDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Routine,
    }),
    RoutineFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    RoutineUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    RunProjectStart: (data) => ({
        //...
    } as any),
    RunRoutineStart: (data) => ({
        //...
    } as any),
    ScheduleAdd: (data) => ({
        id: uuid(),
        timezone: "", //TODO need timezone from user session
        //...
    }),
    ScheduleDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Schedule,
    }),
    ScheduleFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ScheduleUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    SmartContractAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    SmartContractDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Code,
    }),
    SmartContractFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    SmartContractUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    StandardAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    StandardDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Standard,
    }),
    StandardFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    StandardUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
    Start: (data) => ({}),
    TeamAdd: (data) => ({
        id: uuid(),
        isPrivate: true,
        //...
    }),
    TeamDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Team,
    }),
    TeamFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    TeamUpdate: (data) => ({
        id: validUuid(data.id) ?? "",
        //...
    }),
};
