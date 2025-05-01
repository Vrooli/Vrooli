import { BotCreateInput, BotUpdateInput, DEFAULT_LANGUAGE, DeleteManyInput, DeleteOneInput, LlmTask, MemberSearchInput, MemberUpdateInput, ModelType, NavigableObject, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput, ResourceCreateInput, ResourceSearchInput, ResourceUpdateInput, RunCreateInput, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, SessionUser, TeamCreateInput, TeamSearchInput, TeamUpdateInput, User, UserSearchInput, UserTranslation, getObjectSlug, getObjectUrlBase, validatePK } from "@local/shared";
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
import { RecursivePartial } from "../../types.js";
import { processRun } from "../run/queue.js";

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
    BotAdd: ConverterFunc<BotCreateInput>,
    BotDelete: ConverterFunc<DeleteOneInput>,
    BotFind: ConverterFunc<UserSearchInput>,
    BotUpdate: ConverterFunc<BotUpdateInput, Pick<User, "botSettings" | "handle" | "id" | "isBotDepictingPerson" | "isPrivate" | "name"> & { translations: Pick<UserTranslation, "id" | "language" | "bio">[] }>,
    MembersAdd: ConverterFunc<TeamUpdateInput>,
    MembersDelete: ConverterFunc<DeleteManyInput>,
    MembersFind: ConverterFunc<MemberSearchInput>,
    MembersUpdate: ConverterFunc<MemberUpdateInput>,
    ReminderAdd: ConverterFunc<ReminderCreateInput>,
    ReminderDelete: ConverterFunc<DeleteOneInput>,
    ReminderFind: ConverterFunc<ReminderSearchInput>,
    ReminderUpdate: ConverterFunc<ReminderUpdateInput>,
    ResourceAdd: ConverterFunc<ResourceCreateInput>,
    ResourceDelete: ConverterFunc<DeleteOneInput>,
    ResourceFind: ConverterFunc<ResourceSearchInput>,
    ResourceUpdate: ConverterFunc<ResourceUpdateInput>,
    RunStart: ConverterFunc<RunCreateInput>,
    ScheduleAdd: ConverterFunc<ScheduleCreateInput>,
    ScheduleDelete: ConverterFunc<DeleteOneInput>,
    ScheduleFind: ConverterFunc<ScheduleSearchInput>,
    ScheduleUpdate: ConverterFunc<ScheduleUpdateInput>,
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
            validateFields(["id", (data) => validatePK(data.id)])(data);
            const existingUser = await DbProvider.get().user.findUnique({
                where: { id: BigInt(data.id as string) },
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
            const userInput = {
                ...existingUser,
                id: existingUser.id.toString(),
                translations: existingUser.translations.map((t) => ({
                    ...t,
                    id: t.id.toString(),
                })),
            };
            const input = converter[task](data, language, userInput);
            const payload = await Endpoints.user.botUpdateOne({ input }, context, info);
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
            validateFields(["id", (data) => validatePK(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.member.updateOne({ input }, context, info);
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
            validateFields(["id", (data) => validatePK(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.reminder.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ResourceAdd": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/resource.js");
        const info = await import("../../endpoints/generated/resource_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.resource.createOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "ResourceDelete": async ({ context, converter, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/actions.js");
        const info = SuccessInfo;
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await Endpoints.actions.deleteOne({ input }, context, info);
            return { label: null, link: null, payload };
        };
    },
    "ResourceFind": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const info = await import("../../endpoints/generated/resource_findMany.js");
        return async (data) => {
            const input = converter[task](data, language);
            const payload = await readManyWithEmbeddingsHelper({ info, input, objectType: "ResourceVersion", req: context.req });
            const label = payload.edges.length > 0 ? getObjectLabel(payload.edges[0].node) : null;
            const link = payload.edges.length > 0 ? getObjectLink(payload.edges[0].node) : null;
            return { label, link, payload };
        };
    },
    "ResourceUpdate": async ({ context, converter, getObjectLabel, getObjectLink, language, task, validateFields }) => {
        const Endpoints = await import("../../endpoints/logic/resource.js");
        const info = await import("../../endpoints/generated/resource_findOne.js");
        return async (data) => {
            validateFields(["id", (data) => validatePK(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.resource.updateOne({ input }, context, info);
            const label = getObjectLabel(payload);
            const link = getObjectLink(payload);
            return { label, link, payload };
        };
    },
    "RunStart": async ({ context, converter, getObjectLabel, getObjectLink, language, task }) => {
        const Endpoints = await import("../../endpoints/logic/run.js");
        const info = await import("../../endpoints/generated/run_findOne.js");
        return async (data) => {
            const input = converter[task](data, language);
            // Create the run
            const payload = await Endpoints.run.createOne({ input }, context, info);
            // Start the run
            await processRun({} as any);//TODO
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
            validateFields(["id", (data) => validatePK(data.id)])(data);
            const input = converter[task](data, language);
            const payload = await Endpoints.schedule.updateOne({ input }, context, info);
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
            validateFields(["id", (data) => validatePK(data.id)])(data);
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
