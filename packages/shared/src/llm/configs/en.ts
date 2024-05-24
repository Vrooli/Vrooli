import { LlmTask } from "../../api/generated/graphqlTypes";
import { pascalCase } from "../../utils/casing";
import { CommandToTask, LlmTaskConfig, LlmTaskProperty, LlmTaskUnstructuredConfig } from "../types";

export const config: LlmTaskConfig = {
    __suggested_prefix: "suggested",
    __response_formats_with_actions: {
        one_command: "/${command} ${Saction}",
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
    __finish_context_response_format: "/${command} ${action} ${property1}='some_string' ${property2}=123",
    __rules: [
        "Use commands when possible. They are the only way you can perform real actions.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "In general, a command can be used when the user wants to perform an action. For example, if a user asks 'What's the weather?', you can respond with `/routine find`.",
        "When not using a command, you can provide suggested commands at the end of the message. Never suggest more than 4 commands.",
        "When suggesting commands, do not start the suggestion with anything like 'Here are some commands you can use'. Just use the 'suggested_commands' format directly after the message and some whitespace.",
        "Escape single quotes in properties.",
    ],
    __finish_context_rules: [
        "Use a provided command to complete the user's request. This is the only way you can perform real actions.",
        "You must respond with a command.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "Do not provide any other text in the message besides the command.",
        "Escape single quotes in properties.",
    ],
    __pick_properties(selectedFields: [string, boolean | undefined][], __availableFields: Record<string, Omit<LlmTaskProperty, "name">>) {
        return selectedFields.map(([fieldName, isRequired]) => ({
            ...__availableFields[fieldName],
            name: fieldName,
            is_required: isRequired !== undefined ? isRequired : __availableFields[fieldName]?.is_required,
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
        label: undefined, // Label is for displaying the task to the user, and is not needed in the context
    }),
    __construct_context_force: (props: LlmTaskUnstructuredConfig) => ({
        ...config.__construct_context(props),
        _response_format: config.__finish_context_response_format,
        rules: config.__rules,
    }),
    __apiProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    ApiAdd: () => ({
        label: "Add API",
        commands: {
            add: "Create an API with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", true],
            ["description", false],
            //...
        ], config.__apiProperties),
        rules: config.__rules,
    }),
    ApiDelete: () => ({
        label: "Delete API",
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
        label: "Find API",
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
        label: "Update API",
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
        bio: {
            example: "Hi, I'm Elon Musk. I'm the CEO of SpaceX and Tesla, and the founder of The Boring Company and Neuralink. I'm also the former CEO of X (f.k.a. Twitter).",
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
        label: "Add Bot",
        commands: {
            add: "Create a bot with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", true],
            ["bio", false],
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
        label: "Delete Bot",
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
        label: "Find Bot",
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
        label: "Update Bot",
        commands: {
            add: "Update a bot with the provided properties.",
        },
        properties: config.__pick_properties([
            ["id", true],
            ["name", false],
            ["bio", false],
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
        label: "Add Members",
        commands: {
            add: "Create member with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__memberProperties),
        rules: config.__rules,
    }),
    MembersDelete: () => ({
        label: "Delete Members",
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
        label: "Find Members",
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
        label: "Update Members",
        commands: {
            update: "Update a member with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__memberProperties),
        rules: config.__rules,
    }),
    __noteProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    NoteAdd: () => ({
        label: "Add Note",
        commands: {
            add: "Create a note with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__noteProperties),
        rules: config.__rules,
    }),
    NoteDelete: () => ({
        label: "Delete Note",
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
        label: "Find Note",
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
        label: "Update Note",
        commands: {
            update: "Update a note with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__noteProperties),
        rules: config.__rules,
    }),
    __projectProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    ProjectAdd: () => ({
        label: "Add Project",
        commands: {
            add: "Create a project with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__projectProperties),
        rules: config.__rules,
    }),
    ProjectDelete: () => ({
        label: "Delete Project",
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
        label: "Find Project",
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
        label: "Update Project",
        commands: {
            update: "Update a project with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__projectProperties),
        rules: config.__rules,
    }),
    __reminderProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    ReminderAdd: () => ({
        label: "Add Reminder",
        commands: {
            add: "Create a reminder with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__reminderProperties),
        rules: config.__rules,
    }),
    ReminderDelete: () => ({
        label: "Delete Reminder",
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
        label: "Find Reminder",
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
        label: "Update Reminder",
        commands: {
            update: "Update a reminder with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__reminderProperties),
        rules: config.__rules,
    }),
    __roleProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        teamId: {
            description: "The ID of the team to connect the role to",
            type: "uuid",
        },
        //...
    },
    RoleAdd: () => ({
        label: "Add Role",
        commands: {
            add: "Create a role with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__roleProperties),
        rules: config.__rules,
    }),
    RoleDelete: () => ({
        label: "Delete Role",
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
        label: "Find Role",
        commands: {
            find: "Look for existing roles.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
            {
                name: "teamId",
                ...config.__roleProperties.teamId,
                is_required: true,
            },
        ],
        rules: [
            ...config.__rules,
            "Must include at least one search parameter (property).",
        ],
    }),
    RoleUpdate: () => ({
        label: "Update Role",
        commands: {
            update: "Update a role with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__roleProperties),
        rules: config.__rules,
    }),
    __routineProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        // TODO continue
    },
    RoutineAdd: () => ({
        label: "Add Routine",
        commands: {
            add: "Create a routine with the provided properties.",
        },
        properties: config.__pick_properties([
            ["name", true],
            ["description", false],
            //...
        ], config.__routineProperties),
        rules: config.__rules,
    }),
    RoutineDelete: () => ({
        label: "Delete Routine",
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
        label: "Find Routine",
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
        label: "Update Routine",
        commands: {
            update: "Update a routine with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__routineProperties),
        rules: config.__rules,
    }),
    __scheduleProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    ScheduleAdd: () => ({
        label: "Add Schedule",
        commands: {
            add: "Create a schedule with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__scheduleProperties),
        rules: config.__rules,
    }),
    ScheduleDelete: () => ({
        label: "Delete Schedule",
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
        label: "Find Schedule",
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
        label: "Update Schedule",
        commands: {
            update: "Update a schedule with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__scheduleProperties),
        rules: config.__rules,
    }),
    __smartContractProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    SmartContractAdd: () => ({
        label: "Add Smart Contract",
        commands: {
            add: "Create a smart contract with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__smartContractProperties),
        rules: config.__rules,
    }),
    SmartContractDelete: () => ({
        label: "Delete Smart Contract",
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
        label: "Find Smart Contract",
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
        label: "Update Smart Contract",
        commands: {
            update: "Update a smart contract with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__smartContractProperties),
        rules: config.__rules,
    }),
    __standardProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    StandardAdd: () => ({
        label: "Add Standard",
        commands: {
            add: "Create a standard with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__standardProperties),
        rules: config.__rules,
    }),
    StandardDelete: () => ({
        label: "Delete Standard",
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
        label: "Find Standard",
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
        label: "Update Standard",
        commands: {
            update: "Update a standard with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__standardProperties),
        rules: config.__rules,
    }),
    Start: () => ({
        label: "Start",
        actions: ["add", "find", "update", "delete"],
        commands: {
            note: "Store information that you want to remember",
            reminder: "Remind you of something at a specific time, or act as a checklist",
            schedule: "Schedule an event, such as a routine",
            routine: "Complete a series of tasks, either through automation or manual completion",
            project: "Organize notes, routines, projects, apis, smart contracts, standards, and organizations",
            organization: "Own the same types of data as users/bots, and come with a team of members with group messaging",
            role: "Define permissions for members (users/bots) in organizations",
            bot: "Customized AI assistant",
            user: "Another user on the platform. Only allowed to use 'find' action for users.",
            standard: "Define data structure or LLM prompt. Allow for interoperability between subroutines and other applications",
            api: "Connect to other applications",
            smart_contract: "Define a trustless agreement",
        },
        rules: [
            "You are allowed to answer general questions which are not related to the Vrooli platform.",
            ...config.__rules,
        ],
    }),
    __teamProperties: {
        name: {
            example: "WeatherAPI",
        },
        description: {
            example: "Provides current weather information for a given location.",
        },
        //...
    },
    TeamAdd: () => ({
        label: "Add Team",
        commands: {
            add: "Create a team with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__teamProperties),
        rules: config.__rules,
    }),
    TeamDelete: () => ({
        label: "Delete Team",
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
        label: "Find Team",
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
        label: "Update Team",
        commands: {
            update: "Update a team with the provided properties.",
        },
        properties: config.__pick_properties([
            //...
        ], config.__teamProperties),
        rules: config.__rules,
    }),
};

export const commandToTask: CommandToTask = (command, action) => {
    let result: string;
    if (action) result = `${pascalCase(command)}${pascalCase(action)}`;
    else result = pascalCase(command);
    if (Object.keys(LlmTask).includes(result)) return result as LlmTask;
    return null;
};