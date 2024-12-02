import { ApiCreateInput, ApiSearchInput, ApiUpdateInput, BotCreateInput, BotUpdateInput, CodeCreateInput, CodeSearchInput, CodeUpdateInput, DEFAULT_LANGUAGE, DeleteManyInput, DeleteOneInput, GqlModelType, LlmTask, MemberSearchInput, MemberUpdateInput, NavigableObject, NoteCreateInput, NoteSearchInput, NoteUpdateInput, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput, QuestionCreateInput, QuestionSearchInput, QuestionUpdateInput, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput, RoleCreateInput, RoleSearchInput, RoleUpdateInput, RoutineCreateInput, RoutineSearchInput, RoutineUpdateInput, RunProjectCreateInput, RunRoutineCreateInput, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, SessionUser, StandardCreateInput, StandardSearchInput, StandardUpdateInput, TeamCreateInput, TeamSearchInput, TeamUpdateInput, ToBotSettingsPropBot, UserSearchInput, getObjectSlug, getObjectUrlBase, uuidValidate } from "@local/shared";
import { Request, Response } from "express";
import { GraphQLResolveInfo } from "graphql";
import path from "path";
import { fileURLToPath } from "url";
import { readManyWithEmbeddingsHelper } from "../../actions/reads";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events";
import { CustomError } from "../../events/error";
import { Context } from "../../middleware/context";
import { ModelMap } from "../../models/base";
import { processRunProject, processRunRoutine } from "../../tasks/run";
import { CreateOneResult, FindOneResult, UpdateOneResult } from "../../types";

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
    DataConverterAdd: ConverterFunc<CodeCreateInput>,
    DataConverterDelete: ConverterFunc<DeleteOneInput>,
    DataConverterFind: ConverterFunc<CodeSearchInput>,
    DataConverterUpdate: ConverterFunc<CodeUpdateInput>,
    MembersAdd: ConverterFunc<TeamUpdateInput>,
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
    QuestionAdd: ConverterFunc<QuestionCreateInput>,
    QuestionDelete: ConverterFunc<DeleteOneInput>,
    QuestionFind: ConverterFunc<QuestionSearchInput>,
    QuestionUpdate: ConverterFunc<QuestionUpdateInput>,
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
    RunProjectStart: ConverterFunc<RunProjectCreateInput>,
    RunRoutineStart: ConverterFunc<RunRoutineCreateInput>,
    ScheduleAdd: ConverterFunc<ScheduleCreateInput>,
    ScheduleDelete: ConverterFunc<DeleteOneInput>,
    ScheduleFind: ConverterFunc<ScheduleSearchInput>,
    ScheduleUpdate: ConverterFunc<ScheduleUpdateInput>,
    SmartContractAdd: ConverterFunc<CodeCreateInput>,
    SmartContractDelete: ConverterFunc<DeleteOneInput>,
    SmartContractFind: ConverterFunc<CodeSearchInput>,
    SmartContractUpdate: ConverterFunc<CodeUpdateInput>,
    StandardAdd: ConverterFunc<StandardCreateInput>,
    StandardDelete: ConverterFunc<DeleteOneInput>,
    StandardFind: ConverterFunc<StandardSearchInput>,
    StandardUpdate: ConverterFunc<StandardUpdateInput>,
    Start: ConverterFunc<unknown>,
    TeamAdd: ConverterFunc<TeamCreateInput>,
    TeamDelete: ConverterFunc<DeleteOneInput>,
    TeamFind: ConverterFunc<TeamSearchInput>,
    TeamUpdate: ConverterFunc<TeamUpdateInput>,
};

/**
 * Data returned by a task
 */
type LlmTaskResult = {
    /**
    * Label for result of task, if applicable. 
    * 
    * For example, if the task is to create a note, the resultLabel could be
    * the note's title.
    */
    label: string | null;
    /**
     * Link to open the result of the task, if applicable.
     * 
     * For example, if the task is to create a note, the resultLink could be
     * the link to view the note.
     */
    link: string | null;
    /**
     * The full result of the task, if applicable.
     * 
     * For example, if creating a note, this would be the full note object.
     */
    payload?: unknown;
};

/**
 * Wrapper function for executing a task
 */
type LlmTaskExec = (data: LlmTaskData) => (LlmTaskResult | Promise<LlmTaskResult>);

type ValidateFieldsFunc = (...validators: [string, ((data: LlmTaskData) => boolean)][]) => (data: LlmTaskData) => void;
type GetObjectLabelFunc<T extends { __typename: string }> = (object: CreateOneResult<T> | UpdateOneResult<T> | FindOneResult<T>) => string | null;
type GetObjectLinkFunc<T extends object> = (object: CreateOneResult<T> | UpdateOneResult<T> | FindOneResult<T>) => string | null;
type TaskHandlerHelperFuncs<Task extends Exclude<LlmTask, "Start">> = {
    context: Context,
    converter: LlmTaskConverters,
    language: string,
    getObjectLabel: GetObjectLabelFunc<{ __typename: string }>,
    getObjectLink: GetObjectLinkFunc<object>,
    task: Task,
    userData: SessionUser,
    validateFields: ValidateFieldsFunc,
};

const dirname = path.dirname(fileURLToPath(import.meta.url));
export const LLM_CONVERTER_LOCATION = `${dirname}/converters`;

/**
 * Dynamically imports the converter for the specified language.
 */
export async function importConverter(language: string): Promise<LlmTaskConverters> {
    try {
        const { convert } = await import(`${LLM_CONVERTER_LOCATION}/${language}`);
        return convert;
    } catch (error) {
        logger.error(`Converter for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0040" });
        const { convert } = await import(`${LLM_CONVERTER_LOCATION}/${DEFAULT_LANGUAGE}`);
        return convert;
    }
}

/**
 * Dynamically imports a "select" shape from a module based on the module name.
 * @param moduleName The name of the module to import
 * @returns The exported shape from the module
 */
async function loadInfo(moduleName: string): Promise<GraphQLResolveInfo> {
    // Construct the path to the module based on the provided module name
    const path = `../../endpoints/generated/${moduleName}`;

    // Dynamically import the module
    const module = await import(path);

    // Access the export using the same name as the module name
    const exportedValue = module[moduleName] as unknown as GraphQLResolveInfo;

    return exportedValue;
}

const SuccessInfo = { __typename: "Success" as const, success: true } as unknown as GraphQLResolveInfo;
const CountInfo = { __typename: "Count" as const, count: true } as unknown as GraphQLResolveInfo;

const taskHandlerMap: { [Task in Exclude<LlmTask, "Start">]: (helperFuncs: TaskHandlerHelperFuncs<Task>) => Promise<LlmTaskExec> } = {
    "ApiAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { ApiEndpoints } = await import("../../endpoints/logic/api");
        const info = await loadInfo("api_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await ApiEndpoints.Mutation.apiCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ApiDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ApiFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("api_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "ApiVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
        //TODO find tasks will typically have follow-up actions, like picking one of the results or finding more. This means we should probably be running a routine instead
    },
    "ApiUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { ApiEndpoints } = await import("../../endpoints/logic/api");
        const info = await loadInfo("api_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await ApiEndpoints.Mutation.apiUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "BotAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { UserEndpoints } = await import("../../endpoints/logic/user");
        const info = await loadInfo("user_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await UserEndpoints.Mutation.botCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "BotDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "BotFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("user_findMany");
        return async (data) => {
            const input = { ...converter[task](data, language), isBot: true };
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "User", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "BotUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, userData, validateFields }) => {
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
                throw new CustomError("0276", "NotFound", { id: data.id, task });
            }
            const input = converter[task](data, language, existingUser);
            const payload = await UserEndpoints.Mutation.botUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "DataConverterAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { CodeEndpoints } = await import("../../endpoints/logic/code");
        const info = await loadInfo("code_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await CodeEndpoints.Mutation.codeCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "DataConverterDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "DataConverterFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("code_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "CodeVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "DataConverterUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { CodeEndpoints } = await import("../../endpoints/logic/code");
        const info = await loadInfo("code_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await CodeEndpoints.Mutation.codeUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "MembersAdd": async () => {
        // const { MemberEndpoints } = await import("../../endpoints/logic/member");
        // const info = await loadInfo("member_findOne");
        return async (data) => {
            //TODO
            return { label: null, link: null, payload: null };
        };
    },
    "MembersDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = CountInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteMany(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "MembersFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { MemberEndpoints } = await import("../../endpoints/logic/member");
        const info = await loadInfo("member_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await MemberEndpoints.Query.members(undefined, { input }, context, info);
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "MembersUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { MemberEndpoints } = await import("../../endpoints/logic/member");
        const info = await loadInfo("member_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await MemberEndpoints.Mutation.memberUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "NoteAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { NoteEndpoints } = await import("../../endpoints/logic/note");
        const info = await loadInfo("note_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await NoteEndpoints.Mutation.noteCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "NoteDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "NoteFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("note_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "NoteVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "NoteUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { NoteEndpoints } = await import("../../endpoints/logic/note");
        const info = await loadInfo("note_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await NoteEndpoints.Mutation.noteUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ProjectAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { ProjectEndpoints } = await import("../../endpoints/logic/project");
        const info = await loadInfo("project_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await ProjectEndpoints.Mutation.projectCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ProjectDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ProjectFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("project_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "ProjectVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "ProjectUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { ProjectEndpoints } = await import("../../endpoints/logic/project");
        const info = await loadInfo("project_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await ProjectEndpoints.Mutation.projectUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "QuestionAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { QuestionEndpoints } = await import("../../endpoints/logic/question");
        const info = await loadInfo("question_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await QuestionEndpoints.Mutation.questionCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "QuestionDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "QuestionFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("question_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "Question", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "QuestionUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { QuestionEndpoints } = await import("../../endpoints/logic/question");
        const info = await loadInfo("question_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await QuestionEndpoints.Mutation.questionUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ReminderAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task, userData }) => {
        const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
        const info = await loadInfo("reminder_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            if (!input.reminderListConnect && !input.reminderListCreate) {
                const activeReminderList = userData?.activeFocusMode?.focusMode?.reminderListId;
                if (activeReminderList) {
                    input.reminderListConnect = activeReminderList;
                } else {
                    throw new CustomError("0555", "InternalError", { task, data, language });
                }
            }
            const payload = await ReminderEndpoints.Mutation.reminderCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ReminderDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ReminderFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
        const info = await loadInfo("reminder_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await ReminderEndpoints.Query.reminders(undefined, { input }, context, info);
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "ReminderUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { ReminderEndpoints } = await import("../../endpoints/logic/reminder");
        const info = await loadInfo("reminder_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await ReminderEndpoints.Mutation.reminderUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoleAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { RoleEndpoints } = await import("../../endpoints/logic/role");
        const info = await loadInfo("role_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await RoleEndpoints.Mutation.roleCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoleDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "RoleFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { RoleEndpoints } = await import("../../endpoints/logic/role");
        const info = await loadInfo("role_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await RoleEndpoints.Query.roles(undefined, { input }, context, info);
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "RoleUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { RoleEndpoints } = await import("../../endpoints/logic/role");
        const info = await loadInfo("role_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await RoleEndpoints.Mutation.roleUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoutineAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
        const info = await loadInfo("routine_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await RoutineEndpoints.Mutation.routineCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoutineDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "RoutineFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("routine_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "RoutineVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "RoutineUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { RoutineEndpoints } = await import("../../endpoints/logic/routine");
        const info = await loadInfo("routine_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await RoutineEndpoints.Mutation.routineUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RunProjectStart": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { RunProjectEndpoints } = await import("../../endpoints/logic/runProject");
        const info = await loadInfo("runProject_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            // Create the run
            const payload = await RunProjectEndpoints.Mutation.runProjectCreate(undefined, { input }, context, info);
            // Start the run
            await processRunProject({} as any);//TODO
            // Return the run
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RunRoutineStart": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { RunRoutineEndpoints } = await import("../../endpoints/logic/runRoutine");
        const info = await loadInfo("runRoutine_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            // Create the run
            const payload = await RunRoutineEndpoints.Mutation.runRoutineCreate(undefined, { input }, context, info);
            // Start the run
            await processRunRoutine({} as any);//TODO
            // Return the run
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ScheduleAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
        const info = await loadInfo("schedule_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await ScheduleEndpoints.Mutation.scheduleCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ScheduleDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ScheduleFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
        const info = await loadInfo("schedule_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await ScheduleEndpoints.Query.schedules(undefined, { input }, context, info);
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "ScheduleUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { ScheduleEndpoints } = await import("../../endpoints/logic/schedule");
        const info = await loadInfo("schedule_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await ScheduleEndpoints.Mutation.scheduleUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "SmartContractAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { CodeEndpoints } = await import("../../endpoints/logic/code");
        const info = await loadInfo("code_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await CodeEndpoints.Mutation.codeCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "SmartContractDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "SmartContractFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("code_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "CodeVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "SmartContractUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { CodeEndpoints } = await import("../../endpoints/logic/code");
        const info = await loadInfo("code_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await CodeEndpoints.Mutation.codeUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "StandardAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { StandardEndpoints } = await import("../../endpoints/logic/standard");
        const info = await loadInfo("standard_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await StandardEndpoints.Mutation.standardCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "StandardDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "StandardFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("standard_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "StandardVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "StandardUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { StandardEndpoints } = await import("../../endpoints/logic/standard");
        const info = await loadInfo("standard_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await StandardEndpoints.Mutation.standardUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "TeamAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const { TeamEndpoints } = await import("../../endpoints/logic/team");
        const info = await loadInfo("team_findOne");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await TeamEndpoints.Mutation.teamCreate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "TeamDelete": async ({ context, converter, language, task }) => {
        const { DeleteOneOrManyEndpoints } = await import("../../endpoints/logic/deleteOneOrMany");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await DeleteOneOrManyEndpoints.Mutation.deleteOne(undefined, { input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "TeamFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await loadInfo("team_findMany");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "Team", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "TeamUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const { TeamEndpoints } = await import("../../endpoints/logic/team");
        const info = await loadInfo("team_findOne");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await TeamEndpoints.Mutation.teamUpdate(undefined, { input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
};

/**
 * Creates the task execution function for a given task type
 */
export async function generateTaskExec<Task extends Exclude<LlmTask, "Start">>(
    task: Task,
    language: string,
    userData: SessionUser,
): Promise<LlmTaskExec> {
    // Import converter, which shapes data for the task
    const converter = await importConverter(language);
    const languages = userData.languages;
    // NOTE: We are skipping some information in this context, such as the response information. 
    // This is typically only used by middleware, so it should hopefully not be necessary for the LLM.
    const context: Context = {
        req: {
            path: `/llmTask/${task}`, // Required for rate limiting
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
    function validateFields(...validators: Parameters<ValidateFieldsFunc>): ReturnType<ValidateFieldsFunc> {
        return (data: LlmTaskData) => {
            for (const [field, validator] of validators) {
                if (!validator(data)) {
                    throw new CustomError("0047", "InvalidArgs", { field, task });
                }
            }
        };
    }

    /** Creates label for a created/updated/found object */
    function getObjectLabel(object: Parameters<GetObjectLabelFunc<{ __typename: string }>>[0]): ReturnType<GetObjectLabelFunc<{ __typename: string }>> {
        if (object === null || object === undefined) {
            return null;
        }
        const { display } = ModelMap.getLogic(["display"], object.__typename as GqlModelType, true);
        return display().label.get(object, languages);
    }

    /** Creates link for a created/updated/found object */
    function getObjectLink(object: Parameters<GetObjectLinkFunc<object>>[0]): ReturnType<GetObjectLinkFunc<object>> {
        if (object === null || object === undefined) {
            return null;
        }
        return `${getObjectUrlBase(object as NavigableObject)}/${getObjectSlug(object as NavigableObject)}`;
    }

    const helperFuncs = {
        context,
        converter,
        language,
        getObjectLabel,
        getObjectLink,
        task,
        userData,
        validateFields,
    } as const;

    if (taskHandlerMap[task]) {
        return taskHandlerMap[task](helperFuncs as TaskHandlerHelperFuncs<Task>);
    } else {
        throw new CustomError("0043", "TaskNotSupported", { task });
    }
}
