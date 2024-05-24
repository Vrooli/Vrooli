import { ApiCreateInput, ApiSearchInput, ApiUpdateInput, BotCreateInput, BotUpdateInput, DEFAULT_LANGUAGE, DeleteManyInput, DeleteOneInput, LlmTask, MemberSearchInput, MemberUpdateInput, NoteCreateInput, NoteSearchInput, NoteUpdateInput, OrganizationCreateInput, OrganizationSearchInput, OrganizationUpdateInput, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput, RoleCreateInput, RoleSearchInput, RoleUpdateInput, RoutineCreateInput, RoutineSearchInput, RoutineUpdateInput, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, SmartContractCreateInput, SmartContractSearchInput, SmartContractUpdateInput, StandardCreateInput, StandardSearchInput, StandardUpdateInput, ToBotSettingsPropBot, UserSearchInput, uuidValidate } from "@local/shared";
import { Request, Response } from "express";
import { GraphQLResolveInfo } from "graphql";
import path from "path";
import { fileURLToPath } from "url";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events";
import { CustomError } from "../../events/error";
import { Context } from "../../middleware/context";
import { SessionUserToken } from "../../types";

type LlmTaskDataValue = string | number | boolean | null;
export type LlmTaskData = Record<string, LlmTaskDataValue>;

/** 
 * Converts data provided to a command into a shape usable by the rest of the server 
 * 
 * E.g. Converting the properties of a "Create" command into a that object 
 * type's CreateInput type, which is then used later to call the corresponding API endpoint.
 */
type ConverterFunc<ShapedData, U = undefined> = U extends undefined
    ? (data: LlmTaskData, language: string) => ShapedData
    : (data: LlmTaskData, language: string, existingData: U) => ShapedData;
export type LlmTaskConverters = {
    ApiAdd: ConverterFunc<ApiCreateInput>,
    ApiDelete: ConverterFunc<DeleteOneInput>,
    ApiFind: ConverterFunc<ApiSearchInput>,
    ApiUpdate: ConverterFunc<ApiUpdateInput>,
    BotAdd: ConverterFunc<BotCreateInput>,
    BotDelete: ConverterFunc<DeleteOneInput>,
    BotFind: ConverterFunc<UserSearchInput>,
    BotUpdate: ConverterFunc<BotUpdateInput, ToBotSettingsPropBot>,
    MembersAdd: ConverterFunc<OrganizationUpdateInput>,
    MembersDelete: ConverterFunc<DeleteManyInput>,
    MembersFind: ConverterFunc<MemberSearchInput>,
    MembersUpdate: ConverterFunc<MemberUpdateInput>,
    NoteAdd: ConverterFunc<NoteCreateInput>,
    NoteDelete: ConverterFunc<DeleteOneInput>,
    NoteFind: ConverterFunc<NoteSearchInput>,
    NoteUpdate: ConverterFunc<NoteUpdateInput>,
    ProjectAdd: ConverterFunc<ProjectCreateInput>,
    ProjectDelete: ConverterFunc<DeleteOneInput>,
    ProjectFind: ConverterFunc<ProjectSearchInput>,
    ProjectUpdate: ConverterFunc<ProjectUpdateInput>,
    ReminderAdd: ConverterFunc<ReminderCreateInput>,
    ReminderDelete: ConverterFunc<DeleteOneInput>,
    ReminderFind: ConverterFunc<ReminderSearchInput>,
    ReminderUpdate: ConverterFunc<ReminderUpdateInput>,
    RoleAdd: ConverterFunc<RoleCreateInput>,
    RoleDelete: ConverterFunc<DeleteOneInput>,
    RoleFind: ConverterFunc<RoleSearchInput>,
    RoleUpdate: ConverterFunc<RoleUpdateInput>,
    RoutineAdd: ConverterFunc<RoutineCreateInput>,
    RoutineDelete: ConverterFunc<DeleteOneInput>,
    RoutineFind: ConverterFunc<RoutineSearchInput>,
    RoutineUpdate: ConverterFunc<RoutineUpdateInput>,
    ScheduleAdd: ConverterFunc<ScheduleCreateInput>,
    ScheduleDelete: ConverterFunc<DeleteOneInput>,
    ScheduleFind: ConverterFunc<ScheduleSearchInput>,
    ScheduleUpdate: ConverterFunc<ScheduleUpdateInput>,
    SmartContractAdd: ConverterFunc<SmartContractCreateInput>,
    SmartContractDelete: ConverterFunc<DeleteOneInput>,
    SmartContractFind: ConverterFunc<SmartContractSearchInput>,
    SmartContractUpdate: ConverterFunc<SmartContractUpdateInput>,
    StandardAdd: ConverterFunc<StandardCreateInput>,
    StandardDelete: ConverterFunc<DeleteOneInput>,
    StandardFind: ConverterFunc<StandardSearchInput>,
    StandardUpdate: ConverterFunc<StandardUpdateInput>,
    Start: ConverterFunc<unknown>,
    TeamAdd: ConverterFunc<OrganizationCreateInput>,
    TeamDelete: ConverterFunc<DeleteOneInput>,
    TeamFind: ConverterFunc<OrganizationSearchInput>,
    TeamUpdate: ConverterFunc<OrganizationUpdateInput>,
}



const dirname = path.dirname(fileURLToPath(import.meta.url));
export const LLM_CONVERTER_LOCATION = `${dirname}/converters`;

/**
 * Dynamically imports the converter for the specified language.
 */
export const importConverter = async (language: string): Promise<LlmTaskConverters> => {
    try {
        const { convert } = await import(`${LLM_CONVERTER_LOCATION}/${language}`);
        return convert;
    } catch (error) {
        logger.error(`Converter for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0040" });
        const { convert } = await import(`${LLM_CONVERTER_LOCATION}/${DEFAULT_LANGUAGE}`);
        return convert;
    }
};

/**
 * Dynamically imports a "select" shape from a module based on the module name.
 * @param moduleName The name of the module to import
 * @returns The exported shape from the module
 */
const loadInfo = async (moduleName: string): Promise<GraphQLResolveInfo> => {
    // Construct the path to the module based on the provided module name
    const path = `../../endpoints/generated/${moduleName}`;

    // Dynamically import the module
    const module = await import(path);

    // Access the export using the same name as the module name
    const exportedValue = module[moduleName] as unknown as GraphQLResolveInfo;

    return exportedValue;
};

const SuccessInfo = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
const CountInfo = { __typename: "Count" as const, count: true } as unknown as GraphQLResolveInfo;

/**
 * Wrapper function for executing a task
 */
type LlmTaskExec = (data: LlmTaskData) => (unknown | Promise<unknown>);

/**
 * Creates the task execution function for a given task type
 */
export const generateTaskExec = async (
    task: LlmTask | `${LlmTask}`,
    language: string,
    userData: SessionUserToken,
): Promise<LlmTaskExec> => {
    const converter = await importConverter(language);
    // NOTE: We are skipping some information in this context, such as the request and response information. 
    // This is typically only used by middleware, so it should hopefully not be necessary for the LLM.
    const context: Context = {
        req: {
            path: `/llmTask/${task}`, // Needed for rate limiting
            session: {
                fromSafeOrigin: true,
                isLoggedIn: true,
                languages: [language],
                users: [userData],
                validToken: true,
            },
        } as unknown as Request,
        res: {} as unknown as Response,
    };

    /** Ensures that required fields are present */
    const validateFields = (...validators: [string, ((data: LlmTaskData) => boolean)][]) => {
        return (data: LlmTaskData) => {
            for (const [field, validator] of validators) {
                if (!validator(data)) {
                    throw new CustomError("0047", "InvalidArgs", userData.languages, { field, task });
                }
            }
        };
    };

    switch (task) {
        case "ApiAdd": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await loadInfo("api_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                ApiEndpoints.Mutation.apiCreate(undefined, { input }, context, info);
            };
        }
        case "ApiDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "ApiFind": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await loadInfo("api_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                ApiEndpoints.Query.apis(undefined, { input }, context, info);
            };
            //TODO find tasks will typically have follow-up actions, like picking one of the results or finding more
        }
        case "ApiUpdate": {
            const { ApiEndpoints } = await import("../../endpoints/logic/api");
            const info = await loadInfo("api_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                ApiEndpoints.Mutation.apiUpdate(undefined, { input }, context, info);
            };
        }
        case "BotAdd": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await loadInfo("user_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                UserEndpoints.Mutation.botCreate(undefined, { input }, context, info);
            };
        }
        case "BotDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "BotFind": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await loadInfo("user_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                UserEndpoints.Query.users(undefined, { input }, context, info);
            };
        }
        case "BotUpdate": {
            const { UserEndpoints } = await import("../../endpoints/logic/user");
            const info = await loadInfo("user_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const existingUser = await prismaInstance.user.findUnique({
                    where: { id: data.id as string },
                    select: {
                        name: true,
                        translations: true,
                        botSettings: true,
                    },
                });
                if (!existingUser) {
                    throw new CustomError("0276", "NotFound", userData.languages, { id: data.id, task });
                }
                const input = converter[task](data, language, existingUser);
                UserEndpoints.Mutation.botUpdate(undefined, { input }, context, info);
            };
        }
        case "MembersAdd": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await loadInfo("member_findOne");
            return async (data) => {
                //TODO
            };
        }
        case "MembersDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = CountInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteMany(undefined, { input }, context, info);
            };
        }
        case "MembersFind": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await loadInfo("member_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                MemberEndpoints.Query.members(undefined, { input }, context, info);
            };
        }
        case "MembersUpdate": {
            const { MemberEndpoints } = await import("../../endpoints/logic/member");
            const info = await loadInfo("member_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                MemberEndpoints.Mutation.memberUpdate(undefined, { input }, context, info);
            };
        }
        case "NoteAdd": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await loadInfo("note_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                NoteEndpoints.Mutation.noteCreate(undefined, { input }, context, info);
            };
        }
        case "NoteDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "NoteFind": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await loadInfo("note_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                NoteEndpoints.Query.notes(undefined, { input }, context, info);
            };
        }
        case "NoteUpdate": {
            const { NoteEndpoints } = await import("../../endpoints/logic/note");
            const info = await loadInfo("note_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                NoteEndpoints.Mutation.noteUpdate(undefined, { input }, context, info);
            };
        }
        case "ProjectAdd": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await loadInfo("project_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                ProjectEndpoints.Mutation.projectCreate(undefined, { input }, context, info);
            };
        }
        case "ProjectDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "ProjectFind": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await loadInfo("project_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                ProjectEndpoints.Query.projects(undefined, { input }, context, info);
            };
        }
        case "ProjectUpdate": {
            const { ProjectEndpoints } = await import("../../endpoints/logic/project");
            const info = await loadInfo("project_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                ProjectEndpoints.Mutation.projectUpdate(undefined, { input }, context, info);
            };
        }
        case "ReminderAdd": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await loadInfo("reminder_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                if (!input.reminderListConnect && !input.reminderListCreate) {
                    const activeReminderList = userData?.activeFocusMode?.mode?.reminderList?.id;
                    if (activeReminderList) {
                        input.reminderListConnect = activeReminderList;
                    } else {
                        throw new CustomError("0555", "InternalError", userData.languages, { task, data, language });
                    }
                }
                ReminderEndpoints.Mutation.reminderCreate(undefined, { input }, context, info);
            };
        }
        case "ReminderDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "ReminderFind": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await loadInfo("reminder_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                ReminderEndpoints.Query.reminders(undefined, { input }, context, info);
            };
        }
        case "ReminderUpdate": {
            const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
            const info = await loadInfo("reminder_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                ReminderEndpoints.Mutation.reminderUpdate(undefined, { input }, context, info);
            };
        }
        case "RoleAdd": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await loadInfo("role_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                RoleEndpoints.Mutation.roleCreate(undefined, { input }, context, info);
            };
        }
        case "RoleDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "RoleFind": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await loadInfo("role_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                RoleEndpoints.Query.roles(undefined, { input }, context, info);
            };
        }
        case "RoleUpdate": {
            const { RoleEndpoints } = await import("../../endpoints/logic/role");
            const info = await loadInfo("role_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                RoleEndpoints.Mutation.roleUpdate(undefined, { input }, context, info);
            };
        }
        case "RoutineAdd": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await loadInfo("routine_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                RoutineEndpoints.Mutation.routineCreate(undefined, { input }, context, info);
            };
        }
        case "RoutineDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "RoutineFind": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await loadInfo("routine_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                RoutineEndpoints.Query.routines(undefined, { input }, context, info);
            };
        }
        case "RoutineUpdate": {
            const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
            const info = await loadInfo("routine_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                RoutineEndpoints.Mutation.routineUpdate(undefined, { input }, context, info);
            };
        }
        case "ScheduleAdd": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await loadInfo("schedule_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                ScheduleEndpoints.Mutation.scheduleCreate(undefined, { input }, context, info);
            };
        }
        case "ScheduleDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "ScheduleFind": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await loadInfo("schedule_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                ScheduleEndpoints.Query.schedules(undefined, { input }, context, info);
            };
        }
        case "ScheduleUpdate": {
            const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
            const info = await loadInfo("schedule_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                ScheduleEndpoints.Mutation.scheduleUpdate(undefined, { input }, context, info);
            };
        }
        case "SmartContractAdd": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await loadInfo("smartContract_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                SmartContractEndpoints.Mutation.smartContractCreate(undefined, { input }, context, info);
            };
        }
        case "SmartContractDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "SmartContractFind": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await loadInfo("smartContract_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                SmartContractEndpoints.Query.smartContracts(undefined, { input }, context, info);
            };
        }
        case "SmartContractUpdate": {
            const { SmartContractEndpoints } = await import("../../endpoints/logic/smartContract");
            const info = await loadInfo("smartContract_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                SmartContractEndpoints.Mutation.smartContractUpdate(undefined, { input }, context, info);
            };
        }
        case "StandardAdd": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await loadInfo("standard_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                StandardEndpoints.Mutation.standardCreate(undefined, { input }, context, info);
            };
        }
        case "StandardDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "StandardFind": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await loadInfo("standard_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                StandardEndpoints.Query.standards(undefined, { input }, context, info);
            };
        }
        case "StandardUpdate": {
            const { StandardEndpoints } = await import("../../endpoints/logic/standard");
            const info = await loadInfo("standard_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                StandardEndpoints.Mutation.standardUpdate(undefined, { input }, context, info);
            };
        }
        case "TeamAdd": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await loadInfo("organization_findOne");
            return async (data) => {
                const input = converter[task](data, language);
                OrganizationEndpoints.Mutation.organizationCreate(undefined, { input }, context, info);
            };
        }
        case "TeamDelete": {
            const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
            const info = SuccessInfo;
            return async (data) => {
                const input = converter[task](data, language);
                DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            };
        }
        case "TeamFind": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await loadInfo("organization_findMany");
            return async (data) => {
                const input = converter[task](data, language);
                OrganizationEndpoints.Query.organizations(undefined, { input }, context, info);
            };
        }
        case "TeamUpdate": {
            const { OrganizationEndpoints } = await import("../../endpoints/logic/organization");
            const info = await loadInfo("organization_findOne");
            return async (data) => {
                validateFields(["id", (data) => uuidValidate(data.id)])(data);
                const input = converter[task](data, language);
                OrganizationEndpoints.Mutation.organizationUpdate(undefined, { input }, context, info);
            };
        }
        default: {
            throw new CustomError("0043", "TaskNotSupported", userData.languages, { task });
        }
    }
};
