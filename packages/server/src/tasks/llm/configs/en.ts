import { DeleteType, pascalCase, toBotSettings, uuid, uuidValidate } from "@local/shared";
import { noEmptyString, validNumber, validUuid } from "../../../builders/noNull";
import { logger } from "../../../events/logger";
import { CommandToTask, LlmCommandProperty, LlmTask, LlmTaskConfig, LlmTaskConverters, LlmTaskUnstructuredConfig, llmTasks } from "../config";

export const config: LlmTaskConfig = {
    __response_formats_with_actions: {
        one_command: "/${command} ${action}",
        multiple_commands: "/${command} ${action} /${command} ${action}",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command} ${action}, /${command} ${action}]",
    },
    __response_formats_without_actions: {
        one_command: "/${command}",
        multiple_commands: "/${command}, /${command}",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command}, /${command}]",
    },
    __response_formats_with_properties: {
        one_command: "/${command} ${property1}='some_string' ${property2}=123",
        multiple_commands: "/${command} ${property1}=null, /${command} ${property2}='another_string'",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command} ${property1}='${value1}', /${command} ${property2}='${value2}']",
    },
    __response_formats_with_actions_and_properties: {
        one_command: "/${command} ${action} ${property1}='some_string' ${property2}=123",
        multiple_commands: "/${command} ${action} ${property1}=null, /${command} ${action} ${property2}='another_string'",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command} ${action} ${property1}='${value1}', /${command} ${action} ${property2}='${value2}']",
    },
    __rules: [
        "Try to use commands when possible. This is the only way you can perform real actions.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "When using a command, do not use any other modifiers, or provide any other text in the message. For example, if a user wants to remember something, you may responsd with just `/note add` and nothing else.",
        "In general, a command can be used when the user wants to perform an action. For example, if a user asks 'What's the weather?', you can respond with `/routine find`.",
        "When not using a command, you can provide suggested commands at the end of the message. Never suggest more than 4 commands.",
    ],
    __pick_properties(selectedFields: [string, boolean | undefined][], __availableFields: Record<string, Omit<LlmCommandProperty, "name">>) {
        return selectedFields.map(([fieldName, isRequired]) => ({
            ...__availableFields[fieldName],
            name: fieldName,
            is_required: isRequired !== undefined ? isRequired : __availableFields[fieldName].is_required,
        }));
    },
    __construct_context: ({
        actions,
        properties,
        commands,
        ...rest
    }: LlmTaskUnstructuredConfig) => ({
        commands: {
            prefix: "/",
            list: Object.keys(commands),
            descriptions: commands,
        },
        actions: actions ? {
            description: properties ?
                "Placed after `/` and the command, to specify the action to be performed. Can be followed by specified properties. E.g: `/note add name='My Note' content='This is my note'`" :
                "Placed after `/` and the command, to specify the action to be performed. No other modifiers should be applied. E.g: `/note add`",
            list: actions,
        } : undefined,
        properties: properties ? {
            description: actions ?
                "Placed after the action, to specify the properties of the command. E.g: `/note add name='My Note' content='This is my note'`" :
                "Placed after the command, to specify the properties of the command. E.g: `/note name='My Note' content='This is my note'`",
            // Use list if all properties are just a string
            list:
                properties.filter(p => typeof p === "string").length === properties.length ?
                    properties :
                    properties.map(p => ({
                        name: typeof p === "string" ? p : p.name,
                        type: typeof p === "string" ? undefined : p.type,
                        description: typeof p === "string" ? undefined : p.description,
                        example: typeof p === "string" ? undefined : p.example,
                        examples: typeof p === "string" ? undefined : p.examples,
                        is_required: typeof p === "string" ? undefined : p.is_required,
                    })),
        } : undefined,
        _response_formats: actions ? config.__response_formats_with_actions : config.__response_formats_without_actions,
        ...rest,
    }),
    __apiProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        // TODO continue
    },
    ApiAdd: () => ({
        commands: {
            create: "Create an API with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", true],
            ["description", false],
            //...
        ], config.__apiProperties),
        rules: config.__rules,
    }),
    ApiDelete: () => ({
        commands: {
            delete: "Permanently delete an API. Ensure this is intended before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    ApiFind: () => ({
        commands: {
            find: "Search for existing APIs.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",
                examples: ["WeatherAPI", "weather information"],
            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    ApiUpdate: () => ({
        commands: {
            update: "Update an API with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", false],
            ["description", false],
            //...
        ], config.__apiProperties),
        rules: config.__rules,
    }),
    __bot_properties: {
        id: {
            description: "The ID of the bot.",
            type: "uuid",
        },
        name: {
            example: "Elon Musk",
        },
        occupation: {
            example: "CEO, entrepreneur",
        },
        persona: {
            example: "Eccentric, visionary",
        },
        startingMessage: {
            example: "Elon here! How can I help you today?",
        },
        tone: {
            example: "Informal, Friendly",
        },
        keyPhrases: {
            example: "SpaceX, Tesla, X (f.k.a. Twitter), Mars",
        },
        domainKnowledge: {
            example: "Space, electric cars, renewable energy",
        },
        bias: {
            example: "capitalist, libertarian, pro-technology",
        },
        creativity: {
            description: "How creative the bot is. 0-1",
            example: "0.5",
        },
        verbosity: {
            description: "How verbose the bot is. 0-1",
            example: "0.5",
        },
    },
    BotAdd: () => ({
        commands: {
            create: "Create a bot with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", true],
            ["occupation", false],
            ["persona", false],
            ["startingMessage", false],
            ["tone", false],
            ["keyPhrases", false],
            ["domainKnowledge", false],
            ["bias", false],
            ["creativity", false],
            ["verbosity", false],
        ], config.__bot_properties),
        rules: config.__rules,
    }),
    BotDelete: () => ({
        commands: {
            delete: "Permanentely delete a bot. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    BotFind: () => ({
        commands: {
            find: "Look for existing bots.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or occupation.",
                examples: ["Elon Musk", "entrepreneur"],

            },
            {
                name: "memberInOrganizationId",
                type: "uuid",
                is_required: false,
                description: "The ID of the organization the bot is a member of.",
            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    BotUpdate: () => ({
        commands: {
            create: "Update a bot with the provided properties.",
        },
        properties: config.__pick_properties([
            ["id", true],
            ["name", false],
            ["occupation", false],
            ["persona", false],
            ["startingMessage", false],
            ["tone", false],
            ["keyPhrases", false],
            ["domainKnowledge", false],
            ["bias", false],
            ["creativity", false],
            ["verbosity", false],
        ], config.__bot_properties),
        rules: config.__rules,
    }),
    __memberProperties: {
        // TODO
    },
    MembersAdd: () => ({
        commands: {
            create: "Create member with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__memberProperties),
        rules: config.__rules,
    }),
    MembersDelete: () => ({
        commands: {
            delete: "Permanently remove members from a team. Ensure this is intended before proceeding. Note that this will not delete the bot/users themselves.",
        },
        properties: [
            {
                name: "ids",
                type: "uuid array",
            },
        ],
        rules: config.__rules,
    }),
    MembersFind: () => ({
        commands: {
            find: "Look for existing members.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name.",
                examples: ["Elon Musk", "Joe Smith"],

            },
            {
                name: "organizationId",
                type: "uuid",
                is_required: false,
                description: "The ID of the organization the member is a member of.",
            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    MembersUpdate: () => ({
        commands: {
            update: "Update a member with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__memberProperties),
        rules: config.__rules,
    }),
    __noteProperties: {
        // TODO
    },
    NoteAdd: () => ({
        commands: {
            create: "Create a note with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__noteProperties),
        rules: config.__rules,
    }),
    NoteDelete: () => ({
        commands: {
            delete: "Permanentely delete a note. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    NoteFind: () => ({
        commands: {
            find: "Look for existing notes.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    NoteUpdate: () => ({
        commands: {
            update: "Update a note with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__noteProperties),
        rules: config.__rules,
    }),
    __projectProperties: {
        // TODO
    },
    ProjectAdd: () => ({
        commands: {
            create: "Create a project with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__projectProperties),
        rules: config.__rules,
    }),
    ProjectDelete: () => ({
        commands: {
            delete: "Permanentely delete a project. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    ProjectFind: () => ({
        commands: {
            find: "Look for existing projects.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    ProjectUpdate: () => ({
        commands: {
            update: "Update a project with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__projectProperties),
        rules: config.__rules,
    }),
    __reminderProperties: {
        // TODO
    },
    ReminderAdd: () => ({
        commands: {
            create: "Create a reminder with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__reminderProperties),
        rules: config.__rules,
    }),
    ReminderDelete: () => ({
        commands: {
            delete: "Permanentely delete a reminder. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    ReminderFind: () => ({
        commands: {
            find: "Look for existing reminders.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    ReminderUpdate: () => ({
        commands: {
            update: "Update a reminder with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__reminderProperties),
        rules: config.__rules,
    }),
    __roleProperties: {
        // TODO
    },
    RoleAdd: () => ({
        commands: {
            create: "Create a role with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__roleProperties),
        rules: config.__rules,
    }),
    RoleDelete: () => ({
        commands: {
            delete: "Permanentely delete a role. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    RoleFind: () => ({
        commands: {
            find: "Look for existing roles.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    RoleUpdate: () => ({
        commands: {
            update: "Update a role with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__roleProperties),
        rules: config.__rules,
    }),
    __routineProperties: {
        // TODO
    },
    RoutineAdd: () => ({
        commands: {
            create: "Create a routine with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__routineProperties),
        rules: config.__rules,
    }),
    RoutineDelete: () => ({
        commands: {
            delete: "Permanentely delete a routine. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    RoutineFind: () => ({
        commands: {
            find: "Look for existing routines.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    RoutineUpdate: () => ({
        commands: {
            update: "Update a routine with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__routineProperties),
        rules: config.__rules,
    }),
    __scheduleProperties: {
        // TODO
    },
    ScheduleAdd: () => ({
        commands: {
            create: "Create a schedule with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__scheduleProperties),
        rules: config.__rules,
    }),
    ScheduleDelete: () => ({
        commands: {
            delete: "Permanentely delete a schedule. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    ScheduleFind: () => ({
        commands: {
            find: "Look for existing schedules.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    ScheduleUpdate: () => ({
        commands: {
            update: "Update a schedule with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__scheduleProperties),
        rules: config.__rules,
    }),
    __smartContractProperties: {
        // TODO
    },
    SmartContractAdd: () => ({
        commands: {
            create: "Create a smart contract with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__smartContractProperties),
        rules: config.__rules,
    }),
    SmartContractDelete: () => ({
        commands: {
            delete: "Permanentely delete a smart contract. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    SmartContractFind: () => ({
        commands: {
            find: "Look for existing smart contracts.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    SmartContractUpdate: () => ({
        commands: {
            update: "Update a smart contract with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__smartContractProperties),
        rules: config.__rules,
    }),
    __standardProperties: {
        // TODO
    },
    StandardAdd: () => ({
        commands: {
            create: "Create a standard with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__standardProperties),
        rules: config.__rules,
    }),
    StandardDelete: () => ({
        commands: {
            delete: "Permanentely delete a standard. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    StandardFind: () => ({
        commands: {
            find: "Look for existing standards.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    StandardUpdate: () => ({
        commands: {
            update: "Update a standard with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__standardProperties),
        rules: config.__rules,
    }),
    Start: () => ({
        actions: ["add", "find", "update", "delete"],
        commands: {
            note: "Store information that you want to remember. Can add, find, update, and delete.",
            reminder: "Remind you of something at a specific time, or act as a checklist. Can add, find, update, and delete.",
            schedule: "Schedule an event, such as a routine. Can add, find, update, and delete.",
            routine: "Complete a series of tasks, either through automation or manual completion. Can add, find, update, and delete.",
            project: "Organize notes, routines, projects, apis, smart contracts, standards, and organizations. Can add, find, update, and delete.",
            organization: "Own the same types of data as users/bots, and come with a team of members with group messaging. Can add, find, update, and delete.",
            role: "Define permissions for members (users/bots) in organizations. Can add, find, update, and delete.",
            bot: "Customized AI assistant. Can ad, find, update, and delete.",
            user: "Another user on the platform. Can find.",
            standard: "Define data structure or LLM prompt. Allow for interoperability between subroutines and other applications. Can add, find, update, and delete.",
            api: "Connect to other applications. Can add, find, update, and delete.",
            smart_contract: "Define a trustless agreement. Can add, find, update, and delete.",
        },
        rules: config.__rules,
    }),
    __teamProperties: {
        // TODO
    },
    TeamAdd: () => ({
        commands: {
            create: "Create a team with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__teamProperties),
        rules: config.__rules,
    }),
    TeamDelete: () => ({
        commands: {
            delete: "Permanentely delete a team. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
        rules: config.__rules,
    }),
    TeamFind: () => ({
        commands: {
            find: "Look for existing teams.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    TeamUpdate: () => ({
        commands: {
            update: "Update a team with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__teamProperties),
        rules: config.__rules,
    }),
};

export const convert: LlmTaskConverters = {
    ApiAdd: (data) => ({
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

    }),
    BotAdd: (data) => ({
        id: uuid(),
        isBotDepictingPerson: false,
        isPrivate: true,
        name: noEmptyString(data.name) ?? "Bot",
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
        memberInOrganizationId: validUuid(data.memberInOrganizationId),
    }),
    BotUpdate: (data, existing) => {
        const settings = toBotSettings(existing, logger);
        return {
            id: data.id + "",
            name: noEmptyString(data.name),
            botSettings: JSON.stringify({
                translations: {
                    // Add other existing translations
                    en: {
                        occupation: noEmptyString(data.occupation, "existing.asdfasdf"), //TODO finish
                        persona: data.persona,
                        startingMessage: data.startingMessage,
                        tone: data.tone,
                        keyPhrases: data.keyPhrases,
                        domainKnowledge: data.domainKnowledge,
                        bias: data.bias,
                        creativity: data.creativity,
                        verbosity: data.verbosity,
                    },
                },

            }),
        };
    },
    MembersAdd: (data) => ({

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
        organizationId: validUuid(data.organizationId),
    }),
    MembersUpdate: (data) => ({

    }),
    NoteAdd: (data) => ({

    }),
    NoteDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Note,
    }),
    NoteFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    NoteUpdate: (data) => ({

    }),
    ProjectAdd: (data) => ({

    }),
    ProjectDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Project,
    }),
    ProjectFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ProjectUpdate: (data) => ({

    }),
    ReminderAdd: (data) => ({

    }),
    ReminderDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Reminder,
    }),
    ReminderFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ReminderUpdate: (data) => ({

    }),
    RoleAdd: (data) => ({

    }),
    RoleDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Role,
    }),
    RoleFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    RoleUpdate: (data) => ({

    }),
    RoutineAdd: (data) => ({

    }),
    RoutineDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Routine,
    }),
    RoutineFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    RoutineUpdate: (data) => ({

    }),
    ScheduleAdd: (data) => ({

    }),
    ScheduleDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Schedule,
    }),
    ScheduleFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    ScheduleUpdate: (data) => ({

    }),
    SmartContractAdd: (data) => ({

    }),
    SmartContractDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.SmartContract,
    }),
    SmartContractFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    SmartContractUpdate: (data) => ({

    }),
    StandardAdd: (data) => ({

    }),
    StandardDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Standard,
    }),
    StandardFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    StandardUpdate: (data) => ({

    }),
    Start: (data) => ({}),
    TeamAdd: (data) => ({

    }),
    TeamDelete: (data) => ({
        id: validUuid(data.id) ?? "",
        objectType: DeleteType.Organization,
    }),
    TeamFind: (data) => ({
        searchString: noEmptyString(data.searchString),
    }),
    TeamUpdate: (data) => ({

    }),
} as any;

export const commandToTask: CommandToTask = (command, action) => {
    let result: string;
    if (action) result = `${pascalCase(command)}${pascalCase(action)}`;
    else result = pascalCase(command);
    if (llmTasks.includes(result as LlmTask)) return result as LlmTask;
    return null;
};
