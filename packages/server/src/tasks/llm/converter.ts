import { ApiCreateInput, ApiSearchInput, ApiUpdateInput, BotCreateInput, BotUpdateInput, CodeCreateInput, CodeSearchInput, CodeUpdateInput, DEFAULT_LANGUAGE, DeleteManyInput, DeleteOneInput, LlmTask, MemberSearchInput, MemberUpdateInput, ModelType, NavigableObject, NoteCreateInput, NoteSearchInput, NoteUpdateInput, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput, RoleCreateInput, RoleSearchInput, RoleUpdateInput, RoutineCreateInput, RoutineSearchInput, RoutineUpdateInput, RunProjectCreateInput, RunRoutineCreateInput, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, SessionUser, StandardCreateInput, StandardSearchInput, StandardUpdateInput, TeamCreateInput, TeamSearchInput, TeamUpdateInput, User, UserSearchInput, UserTranslation, getObjectSlug, getObjectUrlBase, uuidValidate } from "@local/shared";
import { Request, Response } from "express";
import path from "path";
import { fileURLToPath } from "url";
import { readManyWithEmbeddingsHelper } from "../../actions/reads.js";
import { PartialApiInfo } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Context } from "../../middleware/context.js";
import { ModelMap } from "../../models/base/index.js";
import { processRunProject, processRunRoutine } from "../../tasks/run/queue.js";
import { RecursivePartial } from "../../types.js";

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
    BotUpdate: ConverterFunc<BotUpdateInput, Pick<User, "botSettings" | "handle" | "id" | "isBotDepictingPerson" | "isPrivate" | "name"> & { translations: Pick<UserTranslation, "id" | "language" | "bio">[] }>,
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
type GetObjectLabelFunc<T extends { __typename: string }> = (object: RecursivePartial<T>) => string | null;
type GetObjectLinkFunc<T extends object> = (object: RecursivePartial<T>) => string | null;
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

const SuccessInfo = { __typename: "Success" as const, success: true } as unknown as PartialApiInfo;
const CountInfo = { __typename: "Count" as const, count: true } as unknown as PartialApiInfo;

const taskHandlerMap: { [Task in Exclude<LlmTask, "Start">]: (helperFuncs: TaskHandlerHelperFuncs<Task>) => Promise<LlmTaskExec> } = {
    "ApiAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/api.js");
        const info = await import("../../endpoints/generated/api_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.api.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ApiDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ApiFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/api_findMany.js");
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
        const Endpoints = await import("../../endpoints/logic/api.js");
        const info = await import("../../endpoints/generated/api_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.api.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "BotAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/user.js");
        const info = await import("../../endpoints/generated/user_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.user.botCreateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "BotDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "BotFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/user_findMany.js");
        return async (data) => {
            const input = { ...converter[task](data, language), isBot: true };
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "User", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "BotUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/user.js");
        const info = await import("../../endpoints/generated/user_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const existingUser = await DbProvider.get().user.findUnique({
                where: { id: data.id as string },
                select: {
                    botSettings: true,
                    handle: true,
                    id: true,
                    isBotDepictingPerson: true,
                    isPrivate: true,
                    name: true,
                    translations: {
                        select: {
                            id: true,
                            language: true,
                            bio: true,
                        },
                    },
                },
            });
            if (!existingUser) {
                throw new CustomError("0276", "NotFound", { id: data.id, task });
            }
            const input = converter[task](data, language, existingUser);
            const payload = await Endpoints.user.botUpdateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "DataConverterAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/code.js");
        const info = await import("../../endpoints/generated/code_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.code.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "DataConverterDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "DataConverterFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/code_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "CodeVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "DataConverterUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/code.js");
        const info = await import("../../endpoints/generated/code_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.code.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "MembersAdd": async () => {
        // const Endpoints = await import("../../endpoints/logic/member.js");
        // const info = await import("../../endpoints/generated/member_findOne.js");
        return async (data) => {
            //TODO
            return { label: null, link: null, payload: null };
        };
    },
    "MembersDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = CountInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteMany({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "MembersFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/member.js");
        const info = await import("../../endpoints/generated/member_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.member.findMany({ input }, context, info);
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const label = payload.edges!.length > 0 ? getObjectLabel(payload.edges![0]!.node!) : null;
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const link = payload.edges!.length > 0 ? getObjectLink(payload.edges![0]!.node!) : null;
            return { label, link, payload };
        };
    },
    "MembersUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/member.js");
        const info = await import("../../endpoints/generated/member_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.member.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "NoteAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/note.js");
        const info = await import("../../endpoints/generated/note_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.note.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "NoteDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "NoteFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/note_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "NoteVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "NoteUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/note.js");
        const info = await import("../../endpoints/generated/note_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.note.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ProjectAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/project.js");
        const info = await import("../../endpoints/generated/project_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.project.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ProjectDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ProjectFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/project_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "ProjectVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "ProjectUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/project.js");
        const info = await import("../../endpoints/generated/project_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.project.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ReminderAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task, userData }) => {
        const Endpoints = await import("../../endpoints/logic/reminder.js");
        const info = await import("../../endpoints/generated/reminder_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            if (!input.reminderListConnect && !input.reminderListCreate) {
                throw new CustomError("0555", "InternalError", { task, data, language });
            }
            const payload = await Endpoints.reminder.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ReminderDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ReminderFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/reminder.js");
        const info = await import("../../endpoints/generated/reminder_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.reminder.findMany({ input }, context, info);
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const label = payload.edges!.length > 0 ? getObjectLabel(payload.edges![0]!.node!) : null;
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const link = payload.edges!.length > 0 ? getObjectLink(payload.edges![0]!.node!) : null;
            return { label, link, payload };
        };
    },
    "ReminderUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/reminder.js");
        const info = await import("../../endpoints/generated/reminder_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.reminder.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoleAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/role.js");
        const info = await import("../../endpoints/generated/role_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.role.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoleDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "RoleFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/role.js");
        const info = await import("../../endpoints/generated/role_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.role.findMany({ input }, context, info);
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const label = payload.edges!.length > 0 ? getObjectLabel(payload.edges![0]!.node!) : null;
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const link = payload.edges!.length > 0 ? getObjectLink(payload.edges![0]!.node!) : null;
            return { label, link, payload };
        };
    },
    "RoleUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/role.js");
        const info = await import("../../endpoints/generated/role_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.role.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoutineAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/routine.js");
        const info = await import("../../endpoints/generated/routine_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.routine.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RoutineDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "RoutineFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/routine_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "RoutineVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "RoutineUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/routine.js");
        const info = await import("../../endpoints/generated/routine_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.routine.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RunProjectStart": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/runProject.js");
        const info = await import("../../endpoints/generated/runProject_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            // Create the run
            const payload = await Endpoints.runProject.createOne({ input }, context, info);
            // Start the run
            await processRunProject({} as any);//TODO
            // Return the run
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RunRoutineStart": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/runRoutine.js");
        const info = await import("../../endpoints/generated/runRoutine_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            // Create the run
            const payload = await Endpoints.runRoutine.createOne({ input }, context, info);
            // Start the run
            await processRunRoutine({} as any);//TODO
            // Return the run
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ScheduleAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/schedule.js");
        const info = await import("../../endpoints/generated/schedule_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.schedule.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ScheduleDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ScheduleFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/schedule.js");
        const info = await import("../../endpoints/generated/schedule_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.schedule.findMany({ input }, context, info);
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const label = payload.edges!.length > 0 ? getObjectLabel(payload.edges![0]!.node!) : null;
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const link = payload.edges!.length > 0 ? getObjectLink(payload.edges![0]!.node!) : null;
            return { label, link, payload };
        };
    },
    "ScheduleUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/schedule.js");
        const info = await import("../../endpoints/generated/schedule_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.schedule.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "SmartContractAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/code.js");
        const info = await import("../../endpoints/generated/code_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.code.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "SmartContractDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "SmartContractFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/code_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "CodeVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "SmartContractUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/code.js");
        const info = await import("../../endpoints/generated/code_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.code.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "StandardAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/standard.js");
        const info = await import("../../endpoints/generated/standard_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.standard.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "StandardDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "StandardFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/standard_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "StandardVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "StandardUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/standard.js");
        const info = await import("../../endpoints/generated/standard_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.standard.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "TeamAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/team.js");
        const info = await import("../../endpoints/generated/team_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.team.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "TeamDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "TeamFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/team_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "Team", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "TeamUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/team.js");
        const info = await import("../../endpoints/generated/team_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => uuidValidate(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.team.updateOne({ input }, context, info);
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
        const { display } = ModelMap.getLogic(["display"], object.__typename as ModelType, true);
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
