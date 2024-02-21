import { ApiCreateInput, ApiSearchInput, ApiUpdateInput, BotCreateInput, BotSettings, BotUpdateInput, DeleteManyInput, DeleteOneInput, MemberSearchInput, MemberUpdateInput, NoteCreateInput, NoteSearchInput, NoteUpdateInput, OrganizationCreateInput, OrganizationSearchInput, OrganizationUpdateInput, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput, RoleCreateInput, RoleSearchInput, RoleUpdateInput, RoutineCreateInput, RoutineSearchInput, RoutineUpdateInput, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, SmartContractCreateInput, SmartContractSearchInput, SmartContractUpdateInput, StandardCreateInput, StandardSearchInput, StandardUpdateInput, UserSearchInput, uuidValidate } from "@local/shared";
import { Request, Response } from "express";
import { GraphQLResolveInfo } from "graphql";
import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events";
import { CustomError } from "../../events/error";
import { Context } from "../../middleware/context";
import { PrismaType, SessionUserToken } from "../../types";

export const DEFAULT_LANGUAGE = "en";

/** All available LLM tasks */
export const llmTasks = [
    "ApiAdd",
    "ApiDelete",
    "ApiFind",
    "ApiUpdate",
    "BotAdd",
    "BotDelete",
    "BotFind",
    "BotUpdate",
    "MembersAdd",
    "MembersDelete",
    "MembersFind",
    "MembersUpdate",
    "NoteAdd",
    "NoteDelete",
    "NoteFind",
    "NoteUpdate",
    "ProjectAdd",
    "ProjectDelete",
    "ProjectFind",
    "ProjectUpdate",
    "ReminderAdd",
    "ReminderDelete",
    "ReminderFind",
    "ReminderUpdate",
    "RoleAdd",
    "RoleDelete",
    "RoleFind",
    "RoleUpdate",
    "RoutineAdd",
    "RoutineDelete",
    "RoutineFind",
    "RoutineUpdate",
    "ScheduleAdd",
    "ScheduleDelete",
    "ScheduleFind",
    "ScheduleUpdate",
    "SmartContractAdd",
    "SmartContractDelete",
    "SmartContractFind",
    "SmartContractUpdate",
    "StandardAdd",
    "StandardDelete",
    "StandardFind",
    "StandardUpdate",
    "Start",
    "TeamAdd",
    "TeamDelete",
    "TeamFind",
    "TeamUpdate",
] as const;

/** All available LLM tasks */
export type LlmTask = typeof llmTasks[number];

type LlmCommandDataValue = string | number | null;
export type LlmCommandData = Record<string, LlmCommandDataValue>;

/** Information about a property provided with a command */
export type LlmCommandProperty = {
    name: string,
    type?: string,
    description?: string,
    example?: string,
    examples?: string[],
    is_required?: boolean
};

/** 
 * Command information, which can be used to validate commands or 
 * converted into a structured command to provide to the LLM as context
 */
export type LlmTaskUnstructuredConfig = {
    actions?: string[];
    properties?: (string | LlmCommandProperty)[],
    commands: Record<string, string>,
} & Record<string, any>;

/** Structured command information, which can be used to provide context to the LLM */
export type LlmTaskStructuredConfig = Record<string, any>;

/**
 * Information about all LLM tasks in a given language, including a function to 
 * convert unstructured task information into structured task information
 */
export type LlmTaskConfig = Record<LlmTask, LlmTaskUnstructuredConfig | ((botSettings: BotSettings) => LlmTaskUnstructuredConfig)> & {
    __construct_context: (data: LlmTaskUnstructuredConfig) => LlmTaskStructuredConfig;
}

/**
 * A map of all available LLM tasks and functions which convert a group of properties into 
 * properly-structured input for the task to be executed. 
 * 
 * For example, this could be converting the properties of a "Create" command into a that object 
 * type's CreateInput type, which is then used later to call the corresponding API endpoint.
 */
export type LlmTaskConverters = Record<LlmTask, ((data: LlmCommandData) => unknown) | ((data: LlmCommandData, existing: Record<string, any>) => unknown)>;

/** Converts a command and optional action to a valid task name, or null if invalid */
export type CommandToTask = (command: string, action?: string | null) => (LlmTask | null);

const dirname = path.dirname(fileURLToPath(import.meta.url));
export const LLM_CONFIG_LOCATION = `${dirname}/configs`;

/**
 * Dynamically imports the configuration for the specified language.
 */
export const importConfig = async (language: string): Promise<LlmTaskConfig> => {
    try {
        const { config } = await import(`${LLM_CONFIG_LOCATION}/${language}`);
        return config;
    } catch (error) {
        logger.error(`Configuration for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0309" });
        const { config } = await import(`${LLM_CONFIG_LOCATION}/${DEFAULT_LANGUAGE}`);
        return config;
    }
}

/**
 * Dynamically imports the converter for the specified language.
 */
export const importConverter = async (language: string): Promise<LlmTaskConverters> => {
    try {
        const { convert } = await import(`${LLM_CONFIG_LOCATION}/${language}`);
        return convert;
    } catch (error) {
        logger.error(`Converter for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0040" });
        const { convert } = await import(`${LLM_CONFIG_LOCATION}/${DEFAULT_LANGUAGE}`);
        return convert;
    }
}

/**
 * Dynamically imports the `commandToTask` function, which converts a command and 
 * action to a task name.
 */
export const importCommandToTask = async (language: string): Promise<CommandToTask> => {
    try {
        const { commandToTask } = await import(`${LLM_CONFIG_LOCATION}/${language}`);
        return commandToTask;
    } catch (error) {
        logger.error(`Command to task function for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0041" });
        const { commandToTask } = await import(`${LLM_CONFIG_LOCATION}/${DEFAULT_LANGUAGE}`);
        return commandToTask;
    }
}

/**
 * @returns The unstructured configuration object for the given task, 
 * in the best language available for the user
 */
export const getUnstructuredTaskConfig = async (
    task: LlmTask,
    botSettings: BotSettings,
    language: string = DEFAULT_LANGUAGE,
): Promise<LlmTaskUnstructuredConfig> => {
    const unstructuredConfig = await importConfig(language);
    const taskConfig = unstructuredConfig[task];

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0305" });
        return {} as LlmTaskUnstructuredConfig;
    }

    return taskConfig(botSettings);
};

/**
 * @returns The structured configuration object for the given task,
 * in the best language available for the user
 */
export const getStructuredTaskConfig = async (
    task: LlmTask,
    botSettings: BotSettings,
    language: string = DEFAULT_LANGUAGE,
): Promise<LlmTaskStructuredConfig> => {
    const unstructuredConfig = await importConfig(language);
    const taskConfig = unstructuredConfig[task];

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0046" });
        return {} as LlmTaskStructuredConfig;
    }

    return unstructuredConfig.__construct_context(taskConfig(botSettings));
}

/**
 * Wrapper function for executing a task
 */
type LlmTaskExec = (data: LlmCommandData, botSettings: BotSettings) => (unknown | Promise<unknown>);

/**
 * Creates the task execution function for a given task type
 */
export const generateTaskExec = async (task: LlmTask, language: string, prisma: PrismaType, userData: SessionUserToken): Promise<LlmTaskExec> => {
    const converter = await importConverter(language);
    // NOTE: We are skipping some information in this context, such as the request and response information. 
    // This is typically only used by middleware, so it should hopefully not be necessary for the LLM.
    // TODO: Realized this won't work for rate limiting. Need solution
    const context: Context = {
        prisma,
        req: {
            session: {
                fromSafeOrigin: true,
                isLoggedIn: true,
                languages: [language],
                users: [userData],
                validToken: true,
            }
        } as unknown as Request,
        res: {} as unknown as Response,
    };

    /** Ensures that required fields are present */
    const validateFields = (...validators: [string, ((data: LlmCommandData) => boolean)][]) => {
        return (data: LlmCommandData) => {
            for (const [field, validator] of validators) {
                if (!validator(data)) {
                    throw new CustomError("0047", "InvalidArgs", userData.languages, { field, task });
                }
            }
        };
    }

    switch (task) {
        case "ApiAdd": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await import("../../endpoints/generated/api_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ApiCreateInput;
                ApiEndpoints.Mutation.apiCreate(undefined, { input }, context, info);
            }
        }
        case "ApiDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "ApiFind": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await import("../../endpoints/generated/api_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ApiSearchInput;
                ApiEndpoints.Query.apis(undefined, { input }, context, info);
            }
        }
        case "ApiUpdate": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await import("../../endpoints/generated/api_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as ApiUpdateInput;
                ApiEndpoints.Mutation.apiUpdate(undefined, { input }, context, info);
            }
        }
        case "BotAdd": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await import("../../endpoints/generated/user_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as BotCreateInput;
                UserEndpoints.Mutation.botCreate(undefined, { input }, context, info);
            }
        }
        case "BotDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "BotFind": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await import("../../endpoints/generated/user_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as UserSearchInput;
                UserEndpoints.Query.users(undefined, { input }, context, info);
            }
        }
        case "BotUpdate": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await import("../../endpoints/generated/user_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const existingUser = await prisma.user.findUnique({
                    where: { id: data.id as string },
                    select: {
                        name: true,
                        translations: true,
                        botSettings: true,
                    }
                });
                if (!existingUser) {
                    throw new CustomError("0276", "NotFound", userData.languages, { id: data.id, task });
                }
                const input = converter[task](data, existingUser) as BotUpdateInput;
                UserEndpoints.Mutation.botUpdate(undefined, { input }, context, info);
            }
        }
        case "MembersAdd": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await import("../../endpoints/generated/member_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                //TODO
            }
        }
        case "MembersDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Count" as const, count: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteManyInput;
                DeleteOneOrManyEndpoints.Mutation.deleteMany(undefined, { input }, context, info);
            }
        }
        case "MembersFind": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await import("../../endpoints/generated/member_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as MemberSearchInput;
                MemberEndpoints.Query.members(undefined, { input }, context, info);
            }
        }
        case "MembersUpdate": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await import("../../endpoints/generated/member_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as MemberUpdateInput;
                MemberEndpoints.Mutation.memberUpdate(undefined, { input }, context, info);
            }
        }
        case "NoteAdd": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await import("../../endpoints/generated/note_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as NoteCreateInput;
                NoteEndpoints.Mutation.noteCreate(undefined, { input }, context, info);
            }
        }
        case "NoteDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "NoteFind": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await import("../../endpoints/generated/note_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as NoteSearchInput;
                NoteEndpoints.Query.notes(undefined, { input }, context, info);
            }
        }
        case "NoteUpdate": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await import("../../endpoints/generated/note_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as NoteUpdateInput;
                NoteEndpoints.Mutation.noteUpdate(undefined, { input }, context, info);
            }
        }
        case "ProjectAdd": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await import("../../endpoints/generated/project_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ProjectCreateInput;
                ProjectEndpoints.Mutation.projectCreate(undefined, { input }, context, info);
            }
        }
        case "ProjectDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "ProjectFind": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await import("../../endpoints/generated/project_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ProjectSearchInput;
                ProjectEndpoints.Query.projects(undefined, { input }, context, info);
            }
        }
        case "ProjectUpdate": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await import("../../endpoints/generated/project_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as ProjectUpdateInput;
                ProjectEndpoints.Mutation.projectUpdate(undefined, { input }, context, info);
            }
        }
        case "ReminderAdd": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await import("../../endpoints/generated/reminder_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ReminderCreateInput;
                ReminderEndpoints.Mutation.reminderCreate(undefined, { input }, context, info);
            }
        }
        case "ReminderDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "ReminderFind": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await import("../../endpoints/generated/reminder_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ReminderSearchInput;
                ReminderEndpoints.Query.reminders(undefined, { input }, context, info);
            }
        }
        case "ReminderUpdate": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await import("../../endpoints/generated/reminder_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as ReminderUpdateInput;
                ReminderEndpoints.Mutation.reminderUpdate(undefined, { input }, context, info);
            }
        }
        case "RoleAdd": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await import("../../endpoints/generated/role_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as RoleCreateInput;
                RoleEndpoints.Mutation.roleCreate(undefined, { input }, context, info);
            }
        }
        case "RoleDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "RoleFind": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await import("../../endpoints/generated/role_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as RoleSearchInput;
                RoleEndpoints.Query.roles(undefined, { input }, context, info);
            }
        }
        case "RoleUpdate": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await import("../../endpoints/generated/role_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as RoleUpdateInput;
                RoleEndpoints.Mutation.roleUpdate(undefined, { input }, context, info);
            }
        }
        case "RoutineAdd": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await import("../../endpoints/generated/routine_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as RoutineCreateInput;
                RoutineEndpoints.Mutation.routineCreate(undefined, { input }, context, info);
            }
        }
        case "RoutineDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "RoutineFind": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await import("../../endpoints/generated/routine_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as RoutineSearchInput;
                RoutineEndpoints.Query.routines(undefined, { input }, context, info);
            }
        }
        case "RoutineUpdate": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await import("../../endpoints/generated/routine_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as RoutineUpdateInput;
                RoutineEndpoints.Mutation.routineUpdate(undefined, { input }, context, info);
            }
        }
        case "ScheduleAdd": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await import("../../endpoints/generated/schedule_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ScheduleCreateInput;
                ScheduleEndpoints.Mutation.scheduleCreate(undefined, { input }, context, info);
            }
        }
        case "ScheduleDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "ScheduleFind": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await import("../../endpoints/generated/schedule_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as ScheduleSearchInput;
                ScheduleEndpoints.Query.schedules(undefined, { input }, context, info);
            }
        }
        case "ScheduleUpdate": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await import("../../endpoints/generated/schedule_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as ScheduleUpdateInput;
                ScheduleEndpoints.Mutation.scheduleUpdate(undefined, { input }, context, info);
            }
        }
        case "SmartContractAdd": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await import("../../endpoints/generated/smartContract_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as SmartContractCreateInput;
                SmartContractEndpoints.Mutation.smartContractCreate(undefined, { input }, context, info);
            }
        }
        case "SmartContractDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "SmartContractFind": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await import("../../endpoints/generated/smartContract_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as SmartContractSearchInput;
                SmartContractEndpoints.Query.smartContracts(undefined, { input }, context, info);
            }
        }
        case "SmartContractUpdate": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await import("../../endpoints/generated/smartContract_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as SmartContractUpdateInput;
                SmartContractEndpoints.Mutation.smartContractUpdate(undefined, { input }, context, info);
            }
        }
        case "StandardAdd": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await import("../../endpoints/generated/standard_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as StandardCreateInput;
                StandardEndpoints.Mutation.standardCreate(undefined, { input }, context, info);
            }
        }
        case "StandardDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "StandardFind": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await import("../../endpoints/generated/standard_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as StandardSearchInput;
                StandardEndpoints.Query.standards(undefined, { input }, context, info);
            }
        }
        case "StandardUpdate": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await import("../../endpoints/generated/standard_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as StandardUpdateInput;
                StandardEndpoints.Mutation.standardUpdate(undefined, { input }, context, info);
            }
        }
        case "TeamAdd": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await import("../../endpoints/generated/organization_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as OrganizationCreateInput;
                OrganizationEndpoints.Mutation.organizationCreate(undefined, { input }, context, info);
            }
        }
        case "TeamDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as DeleteOneInput;
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            }
        }
        case "TeamFind": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await import("../../endpoints/generated/organization_findMany") as unknown as GraphQLResolveInfo;
            return async (data) => {
                const input = converter[task](data, {}) as OrganizationSearchInput;
                OrganizationEndpoints.Query.organizations(undefined, { input }, context, info);
            }
        }
        case "TeamUpdate": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await import("../../endpoints/generated/organization_findOne") as unknown as GraphQLResolveInfo;
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, {}) as OrganizationUpdateInput;
                OrganizationEndpoints.Mutation.organizationUpdate(undefined, { input }, context, info);
            }
        }
        default: {
            throw new CustomError("0043", "TaskNotSupported", userData.languages, { task });
        }
    }
};