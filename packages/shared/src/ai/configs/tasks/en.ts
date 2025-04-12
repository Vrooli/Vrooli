import { AITaskConfig, AITaskConfigBuilder, AITaskProperty, AITaskUnstructuredConfig } from "../../types.js";

export const builder: AITaskConfigBuilder = {
    __pick_properties(selectedFields: [string, boolean | undefined][], __availableFields: Record<string, Omit<AITaskProperty, "name">>) {
        return selectedFields.map(([fieldName, isRequired]) => ({
            name: fieldName,
            is_required: typeof isRequired === "boolean" ? isRequired : __availableFields[fieldName]?.is_required,
            ...__availableFields[fieldName],
        }));
    },
    __construct_context_base: ({
        actions,
        properties,
        commands,
        ...rest
    }: AITaskUnstructuredConfig) => ({
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
            list: properties,
        } : undefined,
        response_formats: actions ? config.__response_formats_with_actions : config.__response_formats_without_actions,
        ...rest,
        // Label is for displaying the task to the user, and is not needed in the context
        label: undefined,
    }),
    __construct_context_text: (props: AITaskUnstructuredConfig) => ({
        ...builder.__construct_context_base(props),
        rules: Array.isArray(props.rules) ? [...(config.__rules_text as string[]), ...props.rules] : config.__rules_text,

    }),
    __construct_context_text_force: (props: AITaskUnstructuredConfig) => ({
        ...builder.__construct_context_base(props),
        response_formats: undefined,
        response_format: config.__finish_context_response_format,
        rules: Array.isArray(props.rules) ? [...(config.__rules_text_force as string[]), ...props.rules] : config.__rules_text,
    }),
    __construct_context_json: (props: AITaskUnstructuredConfig) => {
        const base = builder.__construct_context_base(props);
        // Remove text-specific instructions
        delete base.commands.prefix;
        delete base.actions?.description;
        base.actions = base.actions?.list;
        delete base.properties?.description;
        base.properties = base.properties?.list;
        delete base.response_formats;

        return {
            ...base,
            rules: Array.isArray(props.rules) ? [...(config.__rules_json as string[]), ...props.rules] : (config.__rules_json as string[]),
        };
    },
};

export const config: AITaskConfig = {
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
    __rules_text: [
        "Use commands when possible. They are the only way you can perform real actions.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "In general, a command can be used when the user wants to perform an action. For example, if a user asks 'What's the weather?', you can respond with `/routine find`.",
        "When not using a command, you can provide suggested commands at the end of the message. Never suggest more than 4 commands.",
        "When suggesting commands, do not start the suggestion with anything like 'Here are some commands you can use'. Just use the 'suggested_commands' format directly after the message and some whitespace.",
        "Escape single quotes in properties.",
    ],
    __rules_text_force: [
        "Use a provided command to complete the user's request. This is the only way you can perform real actions.",
        "You must respond with a command.",
        "Do not play pretend. If you cannot do something directly, do not pretend to do it.",
        "Do not provide any other text in the message besides the command.",
        "Escape single quotes in properties.",
    ],
    __rules_json: [
        "Respond only with JSON formatted messages.",
        "JSON objects should contain a required \"command\", optional \"action\", and optional \"properties\".",
        "If you need to provide multiple commands, use an array of objects.",
        "Do not include any extra text outside of the JSON structure.",
        "Ensure JSON keys and string values are properly quoted.",
        "Use only the commands, actions, and properties defined in the configuration for generating responses.",
    ],
    __task_name_map: {
        command: {
            note: "Note",
            reminder: "Reminder",
            schedule: "Schedule",
            routine: "Routine",
            project: "Project",
            team: "Team",
            role: "Role",
            bot: "Bot",
            user: "User",
            standard: "Standard",
            api: "Api",
            smart_contract: "SmartContract",
        },
        action: {
            add: "Add",
            find: "Find",
            update: "Update",
            delete: "Delete",
        },
    },
    __apiProperties: {
        id: {
            description: "Unique identifier for the API.",
            type: "uuid",
        },
        name: {
            description: "The name of the API.",
            example: "WeatherAPI",
        },
        summary: {
            description: "A brief description of what the API does.",
            example: "Provides current weather information for a given location.",
        },
        details: {
            description: "A more detailed description of what the API does.",
            example: "This API provides current weather information for a given location. It includes temperature, humidity, wind speed,... (the rest omitted for brevity, but you get the idea).",
        },
        version: {
            description: "Current version of the API.",
            example: "1.0.3",
        },
        callLink: {
            description: "Endpoint URL for accessing the API.",
            example: "https://api.example.com/weather",
        },
        documentationLink: {
            description: "URL to the API's documentation.",
            example: "https://docs.example.com/api/weather",
        },
        isPrivate: {
            description: "Whether the API is private or publicly accessible.",
            type: "boolean",
            default: true,
        },
        schemaLanguage: {
            description: "The language used to define the API schema. Must be one of the \"examples\" types. \"yaml\" is preferred.",
            type: "enum",
            examples: ["javascript", "graphql", "yaml"],
        },
        schemaText: {
            description: "The schema for the API, in the language specified by \"schemaLanguage\". Prefer writing the schema using the OpenAPI Specification version 3.1.0, unless specified otherwise.",
            type: "string",
        },
    },
    ApiAdd: () => ({
        label: "Add API",
        commands: {
            add: "Create an API with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["summary", true],
            ["details", false],
            ["version", true],
            ["callLink", true],
            ["documentationLink", false],
            ["isPrivate", true],
            ["schemaLanguage", true],
            ["schemaText", true],
        ], config.__apiProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    ApiUpdate: () => ({
        label: "Update API",
        commands: {
            update: "Update an API with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["summary", false],
            ["details", false],
            ["version", false],
            ["callLink", false],
            ["documentationLink", false],
            ["isPrivate", false],
            ["schemaLanguage", false],
            ["schemaText", false],
        ], config.__apiProperties),
    }),
    __bot_properties: {
        id: {
            description: "The ID of the bot.",
            type: "uuid",
        },
        name: {
            example: "Elon Musk",
        },
        isBotDepictingPerson: {
            description: "Indicates whether the bot is depicting a real person.",
            type: "boolean",
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
        properties: builder.__pick_properties([
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
                name: "memberInTeamId",
                type: "uuid",
                is_required: false,
                description: "The ID of the team the bot is a member of.",
            },
        ],
        rules: ["Must include at least one search parameter (property)."],
    }),
    BotUpdate: () => ({
        label: "Update Bot",
        commands: {
            add: "Update a bot with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["isBotDepictingPerson", false],
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
    }),
    __dataConverterProperties: {
        id: {
            description: "Unique identifier for the converter.",
            type: "uuid",
        },
        name: {
            description: "The name of the data converter.",
            example: "String to Number Array",
        },
        description: {
            description: "A brief description of what the data converter does.",
            example: "Converts a string of numbers separated by commas into an array of numbers.",
        },
        version: {
            description: "Current version of the data converter.",
            example: "1.0.3",
        },
        isPrivate: {
            description: "Whether the data converter is private or publicly accessible.",
            type: "boolean",
            default: true,
        },
        content: {
            description: "The JavaScript function, written as a string, that transforms the data from one format to another. Begin with a docstring immediately before the function declaration that includes a detailed description of the function's purpose, its parameters, and its return value. Do not include any imports or require statements. Comments should be clear and concise, placed appropriately to explain non-obvious parts of the code.",
            type: "string",
            example: "/**\n * Converts a comma-separated string of numbers into an array of numbers.\n * @param {string} numbersString - A string containing numbers separated by commas.\n * @return {Array<number>} - An array of numbers extracted from the given string.\n */\nfunction stringToNumberArray(numbersString) {\n  return numbersString.split(',').map(Number);\n}",
        },
    },
    DataConverterAdd: () => ({
        label: "Add Code Converter Function",
        commands: {
            add: "Create a JavaScript function to transform data from one format to another.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", true],
            ["version", true],
            ["isPrivate", true],
            ["content", true],
        ], config.__dataConverterProperties),
    }),
    DataConverterDelete: () => ({
        label: "Delete Code Converter Function",
        commands: {
            delete: "Permanentely delete a code converter function. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
    }),
    DataConverterFind: () => ({
        label: "Find Code Converter Function",
        commands: {
            find: "Look for existing code converter functions.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: ["Must include at least one search parameter (property)."],
    }),
    DataConverterUpdate: () => ({
        label: "Update Code Converter Function",
        commands: {
            update: "Update a code converter function with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
            ["version", false],
            ["isPrivate", false],
            ["content", false],
        ], config.__dataConverterProperties),
    }),
    __memberProperties: {
        id: {
            description: "The unique identifier for the member.",
            type: "uuid",
        },
        isAdmin: {
            description: "Whether the member has administrative privileges.",
            type: "boolean",
            default: false,
        },
        teamId: {
            description: "The ID of the team the member belongs to.",
            type: "uuid",
        },
        userId: {
            description: "The ID of the user associated with this member.",
            type: "uuid",
        },
    },
    __memberInviteProperties: {
        willBeAdmin: {
            description: "Whether the invited member will have administrative privileges.",
            type: "boolean",
            default: false,
        },
        teamId: {
            description: "The ID of the team to which the member is being invited.",
            type: "uuid",
        },
        userId: {
            description: "The ID of the user being invited.",
            type: "uuid",
        },
        message: {
            description: "An optional message to the user being invited.",
            example: "Welcome to our team!",
        },
    },
    MembersAdd: () => ({
        label: "Add Members",
        commands: {
            add: "Invite a member with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["willBeAdmin", false],
            ["willHavePermissions", false],
            ["teamId", true],
            ["userId", true],
            ["message", false],
        ], config.__memberInviteProperties),
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
                name: "teamId",
                type: "uuid",
                is_required: false,
                description: "The ID of the team the member is a member of.",
            },
        ],
        rules: ["Must include at least one search parameter (property)."],
    }),
    MembersUpdate: () => ({
        label: "Update Members",
        commands: {
            update: "Update a member with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["isAdmin", false],
            ["permissions", false],
        ], config.__memberProperties),
    }),
    __noteProperties: {
        id: {
            description: "The ID of the note.",
            type: "uuid",
        },
        name: {
            example: "Weekly Review",
        },
        description: {
            example: "A summary of the week's progress and key points.",
        },
        text: {
            example: "This week, we made significant progress on the WeatherAPI project. We also had a successful meeting with the client...",
        },
        isPrivate: {
            type: "boolean",
            default: true,
        },
    },
    NoteAdd: () => ({
        label: "Add Note",
        commands: {
            add: "Create a note with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", true],
            ["text", true],
            ["isPrivate", false],
        ], config.__noteProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    NoteUpdate: () => ({
        label: "Update Note",
        commands: {
            update: "Update a note with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
            ["text", false],
            ["isPrivate", false],
        ], config.__noteProperties),
    }),
    __projectProperties: {
        id: {
            description: "The unique identifier for the project.",
            type: "uuid",
        },
        name: {
            description: "The name of the project.",
            example: "WeatherAPI",
        },
        description: {
            description: "A brief description of what the project does.",
            example: "Provides current weather information for a given location.",
        },
        isPrivate: {
            description: "Indicates if the project is private.",
            type: "boolean",
            default: true,
        },
    },
    ProjectAdd: () => ({
        label: "Add Project",
        commands: {
            add: "Create a project with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", true],
            ["isPrivate", true],
        ], config.__projectProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    ProjectUpdate: () => ({
        label: "Update Project",
        commands: {
            update: "Update a project with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
            ["isPrivate", false],
        ], config.__projectProperties),
    }),
    __questionProperties: {
        id: {
            description: "Unique identifier for the question.",
            type: "uuid",
        },
        name: {
            description: "The question in under 128 characters.",
            example: "How do I start a business?",
        },
        description: {
            description: "More information about the question, in as much detail as needed.",
            example: "I'm looking to start a business, but I'm not sure where to begin. I have some ideas, but I'm not sure how to validate them or what steps to take next.",
        },
    },
    QuestionAdd: () => ({
        label: "Add Question",
        commands: {
            add: "Create a question with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", true],
        ], config.__questionProperties),
    }),
    QuestionDelete: () => ({
        label: "Delete Question",
        commands: {
            delete: "Permanentely delete a question. Make sure you want to do this before proceeding.",
        },
        properties: [
            {
                name: "id",
                type: "uuid",
            },
        ],
    }),
    QuestionFind: () => ({
        label: "Find Question",
        commands: {
            find: "Look for existing questions.",
        },
        properties: [
            {
                name: "searchString",
                is_required: false,
                description: "A string to search for, such as a name or description.",

            },
        ],
        rules: ["Must include at least one search parameter (property)."],
    }),
    QuestionUpdate: () => ({
        label: "Update Question",
        commands: {
            update: "Update a question with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
        ], config.__questionProperties),
    }),
    __reminderProperties: {
        id: {
            description: "The unique identifier for the reminder.",
            type: "uuid",
        },
        name: {
            example: "Project Deadline",
        },
        description: {
            example: "Reminder for the upcoming project submission deadline.",
        },
        dueDate: {
            description: "The date and time when the reminder is due. If steps are provided, should be at or after the last step's due date.",
            type: "DateTime",
            example: "2024-09-30T17:00:00Z",
        },
        isComplete: {
            description: "Status indicating whether the reminder is completed.",
            type: "boolean",
            default: false,
        },
        steps: {
            description: "The steps to complete the reminder, in a sensible order. These are not strictly required, but strongly recommended for tasks that have multiple steps.",
            type: "array",
            properties: {
                name: {
                    description: "The name of the step.",
                },
                description: {
                    description: "A brief description of the step.",
                },
                dueDate: {
                    description: "The date and time when the step is due.",
                    type: "DateTime",
                    example: "2024-09-28T17:00:00Z",
                },
                isComplete: {
                    description: "Status indicating whether the step is completed.",
                    type: "boolean",
                    default: false,
                },
            },
            example: [
                {
                    name: "Finalize project report",
                    description: "Complete and proofread the final project report.",
                    dueDate: "2024-09-28T17:00:00Z",
                    isComplete: false,
                },
                {
                    name: "Prepare presentation slides",
                    description: "Create and review presentation slides for the project submission.",
                    dueDate: "2024-09-29T17:00:00Z",
                    isComplete: false,
                },
                {
                    name: "Submit project deliverables",
                    description: "Upload all project deliverables to the submission portal.",
                    dueDate: "2024-09-30T16:00:00Z",
                    isComplete: false,
                },
            ],
        },
    },
    ReminderAdd: () => ({
        label: "Add Reminder",
        commands: {
            add: "Create a reminder with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", false],
            ["dueDate", true],
            ["isComplete", true],
            ["steps", false],
        ], config.__reminderProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    ReminderUpdate: () => ({
        label: "Update Reminder",
        commands: {
            update: "Update a reminder with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
            ["dueDate", false],
            ["isComplete", false],
            ["steps", false],
        ], config.__reminderProperties),
    }),
    __roleProperties: {
        id: {
            description: "Unique identifier for the role.",
            type: "uuid",
        },
        name: {
            description: "Name of the role within a team.",
            example: "Data Analyst",
        },
        teamId: {
            description: "The ID of the team the member belongs to.",
            type: "uuid",
        },
    },
    RoleAdd: () => ({
        label: "Add Role",
        commands: {
            add: "Create a role with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["permissions", true],
        ], config.__roleProperties),
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
                ...(config.__roleProperties as { teamId: object }).teamId,
                is_required: true,
            },
        ],
        rules: ["Must include at least one search parameter (property)."],
    }),
    RoleUpdate: () => ({
        label: "Update Role",
        commands: {
            update: "Update a role with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
        ], config.__roleProperties),
    }),
    __routineProperties: {
        id: {
            description: "Unique identifier for the routine.",
            type: "uuid",
        },
        name: {
            description: "The name of the routine.",
            example: "Scientific method",
        },
        isInternal: {
            description: "Whether the routine should be available standalone, or is only part of a specific larger routine. Typically this is false.",
            type: "boolean",
            default: false,
        },
        isPrivate: {
            description: "Whether the routine is private or publicly accessible.",
            type: "boolean",
            default: true,
        },
        routineType: {
            description: "The type of routine. The options are: Informational, MultiStep, Generate, Data, Action, Code, Api, SmartContract.\n- Informational (Basic): No side effects. Used to collect information, provide instructions, or serve as a placeholder.\n- MultiStep: A combination of other routines, defined as a BPMN 2.0 process. Each node in the BPMN diagram corresponds to a routine (of any type), executed in order or concurrently as defined by BPMN gateways.\n- Generate: Sends inputs to an AI model and returns its output.\n- Data: Contains a single hard-coded output (e.g. a string or JSON data).\n- Action: Performs specific actions, such as creating, updating, or deleting objects.\n- Code: Runs code to transform inputs into outputs.\n- API: Sends inputs to an external API and returns the response.\n- SmartContract: Connects to a blockchain smart contract, sending inputs and returning outputs.",
            type: "enum",
            examples: ["Informational", "MultiStep", "Generate", "Data", "Action", "Code", "Api", "SmartContract"],
        },
        bpmnDiagram: {
            description: "The BPMN 2.0 diagram (in XML) for the routine, if it is a MultiStep routine. Each BDSM task should have a name, a corresponding routine type, and a brief description of what it does. Use BPMN constructs like Start Event, End Event, Gateways (if needed), and SequenceFlows.",
            type: "string",
        },
        //... TODO add properties for all routine types
    },
    RoutineAdd: () => ({
        label: "Add Routine",
        commands: {
            add: "Create a routine with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["isInternal", false],
            ["isPrivate", false],
            ["routineType", true],
            ["bpmnDiagram", false],
        ], config.__routineProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    RoutineUpdate: () => ({
        label: "Update Routine",
        commands: {
            update: "Update a routine with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["isInternal", false],
            ["isPrivate", false],
        ], config.__routineProperties),
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
    RunProjectStart: () => ({} as never),
    RunRoutineStart: () => ({} as never),
    ScheduleAdd: () => ({
        label: "Add Schedule",
        commands: {
            add: "Create a schedule with the provided properties.",
        },
        properties: builder.__pick_properties([
            //...
        ], config.__scheduleProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    ScheduleUpdate: () => ({
        label: "Update Schedule",
        commands: {
            update: "Update a schedule with the provided properties.",
        },
        properties: builder.__pick_properties([
            //...
        ], config.__scheduleProperties),
    }),
    __smartContractProperties: {
        id: {
            description: "Unique identifier for the smart contract.",
            type: "uuid",
        },
        name: {
            description: "The name of the smart contract.",
            example: "TokenSwap",
        },
        description: {
            description: "A brief description of what the smart contract does.",
            example: "Facilitates token swaps between two parties on the blockchain.",
        },
        version: {
            description: "Current version of the smart contract.",
            example: "1.0.0",
        },
        isPrivate: {
            description: "Whether the smart contract is private or publicly accessible.",
            type: "boolean",
            default: true,
        },
        codeLanguage: {
            description: "The programming language used for the smart contract.",
            type: "string",
            examples: ["solidity", "haskell"],
            default: "solidity",
        },
        content: {
            description: "The smart contract code, written as a string. If the `codeLanguage` is \"solidity\", generate Solidity code (used by many chains, namely Ethereum). If the `codeLanguage` is \"haskell\", generate Plutus code (used in the Cardano ecosystem). Always use the latest version. Begin with a detailed comment block explaining the contract's purpose, its functions, and any important details. Include appropriate comments throughout the code to explain complex logic or important considerations. Do not include any external imports or dependencies.",
            type: "string",
            example: "pragma solidity ^0.8.0;\n\n/**\n * @title TokenSwap\n * @dev Implements a simple token swap between two parties\n * @notice This contract allows two parties to swap ERC20 tokens\n */\ncontract TokenSwap {\n    // Contract implementation...\n}",
        },
    },
    SmartContractAdd: () => ({
        label: "Add Smart Contract",
        commands: {
            add: "Create a smart contract with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["name", true],
            ["description", true],
            ["version", true],
            ["isPrivate", true],
            ["codeLanguage", true],
            ["content", true],
        ], config.__smartContractProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    SmartContractUpdate: () => ({
        label: "Update Smart Contract",
        commands: {
            update: "Update a smart contract with the provided properties.",
        },
        properties: builder.__pick_properties([
            ["id", true],
            ["name", false],
            ["description", false],
            ["version", false],
            ["isPrivate", false],
            ["codeLanguage", false],
            ["content", false],
        ], config.__smartContractProperties),
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
        properties: builder.__pick_properties([
            //...
        ], config.__standardProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    StandardUpdate: () => ({
        label: "Update Standard",
        commands: {
            update: "Update a standard with the provided properties.",
        },
        properties: builder.__pick_properties([
            //...
        ], config.__standardProperties),
    }),
    Start: () => ({
        label: "Start",
        actions: ["add", "find", "update", "delete"],
        commands: {
            note: "Store information that you want to remember",
            reminder: "Remind you of something at a specific time, or act as a checklist",
            schedule: "Schedule an event, such as a routine",
            routine: "Complete a series of tasks, either through automation or manual completion",
            project: "Organize notes, routines, projects, apis, smart contracts, standards, and teams",
            team: "Own the same types of data as users/bots, and come with a team of members with group messaging",
            role: "Define permissions for members (users/bots) in teams",
            bot: "Customized AI assistant",
            user: "Another user on the platform. Only allowed to use 'find' action for users.",
            standard: "Define data structure or LLM prompt. Allow for interoperability between subroutines and other applications",
            api: "Connect to other applications",
            smart_contract: "Define a trustless agreement",
        },
        rules: ["You are allowed to answer general questions which are not related to the Vrooli platform."],
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
        properties: builder.__pick_properties([
            //...
        ], config.__teamProperties),
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
        rules: ["Must include at least one search parameter (property)."],
    }),
    TeamUpdate: () => ({
        label: "Update Team",
        commands: {
            update: "Update a team with the provided properties.",
        },
        properties: builder.__pick_properties([
            //...
        ], config.__teamProperties),
    }),
};
