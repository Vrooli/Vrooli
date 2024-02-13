
type Property = {
    name: string,
    type?: string,
    description?: string,
    example?: string,
    examples?: string[],
    is_required?: boolean
};

const config = {
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
        one_command: "/${command} ${property1}=${value1} ${property2}=${value2}",
        multiple_commands: "/${command} ${property1}=${value1}, /${command} ${property2}=${value2}",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command}, /${command}]",
    },
    __response_formats_with_actions_and_properties: {
        one_command: "/${command} ${action} ${property1}=${value1} ${property2}=${value2}",
        multiple_commands: "/${command} ${action} ${property1}=${value1}, /${command} ${action} ${property2}=${value2}",
        no_command: "${content}",
        suggested_commands: "suggested: [/${command} ${action} ${property1}=${value1}, /${command} ${action} ${property2}=${value2}]",
    },
    __rules: [
        "Try to use commands when possible. This is the only way you can perform real actions.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "When using a command, do not use any other modifiers, or provide any other text in the message. For example, if a user wants to remember something, you may responsd with just `/note add` and nothing else.",
        "In general, a command can be used when the user wants to perform an action. For example, if a user asks 'What's the weather?', you can respond with `/routine find`.",
        "When not using a command, you can provide suggested commands at the end of the message. Never suggest more than 4 commands.",
    ],
    __pick_properties(selectedFields: [string, boolean | undefined][], __botFields: Record<string, Omit<Property, "name">>) {
        return selectedFields.map(([fieldName, isRequired]) => ({
            ...__botFields[fieldName],
            name: fieldName,
            is_required: isRequired !== undefined ? isRequired : __botFields[fieldName].is_required,
        }));
    },
    __define_commands: ({
        actions,
        properties,
        commands,
    }: {
        actions?: string[],
        properties?: (string | Property)[],
        commands: Record<string, string>,
    }) => ({
        commands: {
            prefix: "/",
            list: Object.keys(commands),
            descriptions: commands
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
    }),
    ApiCreate: () => ({

    }),
    ApiDelete: () => ({

    }),
    ApiFind: () => ({

    }),
    ApiUpdate: () => ({

    }),
    __bot_properties: {
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
    BotCreate: () => ({
        ...config.__define_commands({
            commands: {
                create: "Create a bot with the provided properties."
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
        }),
        rules: config.__rules,
    }),
    BotDelete: () => ({
        ...config.__define_commands({
            commands: {
                delete: "Permanentely delete a bot. Make sure you want to do this before proceeding."
            },
            properties: [
                {
                    name: "id",
                    type: "uuid",
                }
            ],
        }),
        rules: config.__rules,
    }),
    BotFind: () => ({
        ...config.__define_commands({
            commands: {
                find: "Look for an existing bot."
            },
            properties: [
                {
                    name: "searchString",
                    is_required: false,
                    description: "A string to search for the bot, such as the name or occupation.",
                    examples: ["Elon Musk", "entrepreneur"],

                },
                {
                    name: "memberInOrganizationId",
                    type: "uuid",
                    is_required: false,
                    description: "The ID of the organization the bot is a member of.",
                }
            ],
        }),
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    BotUpdate: () => ({
        ...config.__define_commands({
            commands: {
                create: "Update a bot with the provided properties."
            },
            properties: config.__pick_properties([
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
        }),
        rules: config.__rules,
    }),
    MembersCreate: () => ({

    }),
    MembersDelete: () => ({

    }),
    MembersFind: () => ({

    }),
    MembersUpdate: () => ({

    }),
    NoteCreate: () => ({

    }),
    NoteDelete: () => ({

    }),
    NoteFind: () => ({

    }),
    NoteUpdate: () => ({

    }),
    ProjectCreate: () => ({

    }),
    ProjectDelete: () => ({

    }),
    ProjectFind: () => ({

    }),
    ProjectUpdate: () => ({

    }),
    ReminderCreate: () => ({

    }),
    ReminderDelete: () => ({

    }),
    ReminderFind: () => ({

    }),
    ReminderUpdate: () => ({

    }),
    RoleCreate: () => ({

    }),
    RoleDelete: () => ({

    }),
    RoleFind: () => ({

    }),
    RoleUpdate: () => ({

    }),
    RoutineCreate: () => ({

    }),
    RoutineDelete: () => ({

    }),
    RoutineFind: () => ({

    }),
    RoutineUpdate: () => ({

    }),
    ScheduleCreate: () => ({

    }),
    ScheduleDelete: () => ({

    }),
    ScheduleFind: () => ({

    }),
    ScheduleUpdate: () => ({

    }),
    SmartContractCreate: () => ({

    }),
    SmartContractDelete: () => ({

    }),
    SmartContractFind: () => ({

    }),
    SmartContractUpdate: () => ({

    }),
    StandardCreate: () => ({

    }),
    StandardDelete: () => ({

    }),
    StandardFind: () => ({

    }),
    StandardUpdate: () => ({

    }),
    Start: () => ({
        ...config.__define_commands({
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
        }),
        rules: config.__rules,
    }),
    TeamCreate: () => ({

    }),
    TeamDelete: () => ({

    }),
    TeamFind: () => ({

    }),
    TeamUpdate: () => ({

    }),
};

export default config;