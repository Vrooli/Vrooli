import { runsCreate, runsUpdate } from "@shared/validation";
import { CODE, RunSortBy } from "@shared/consts";
import { addSupplementalFields, modelToGraphQL, selectHelper, timeFrameToPrisma, toPartialGraphQLInfo, getSearchStringQueryHelper, combineQueries, permissionsSelectHelper } from "./builder";
import { RunStepModel } from "./runStep";
import { Prisma, run, RunStatus } from "@prisma/client";
import { RunInputModel } from "./runInput";
import { CustomError, genErrorCode, Trigger } from "../events";
import { Run, RunSearchInput, RunCreateInput, RunUpdateInput, RunPermission, Count, RunCompleteInput, RunCancelInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, GraphQLInfo, Validator, Mutater } from "./types";
import { cudHelper } from "./actions";
import { routineValidator } from "./routine";

export const runFormatter = (): FormatConverter<Run, any> => ({
    relationshipMap: {
        __typename: 'Run',
        routine: 'Routine',
        steps: 'RunStep',
        inputs: 'RunInput',
        user: 'User',
    },
    removeSupplementalFields: (partial) => {
        // Add fields needed for notifications when a run is started/completed
        return { ...partial, title: true }
    },
})

export const runSearcher = (): Searcher<RunSearchInput> => ({
    defaultSort: RunSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RunSortBy.DateStartedAsc]: { timeStarted: 'asc' },
            [RunSortBy.DateStartedDesc]: { timeStarted: 'desc' },
            [RunSortBy.DateCompletedAsc]: { timeCompleted: 'asc' },
            [RunSortBy.DateCompletedDesc]: { timeCompleted: 'desc' },
            [RunSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [RunSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [RunSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [RunSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    {
                        routine: {
                            translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } },
                        }
                    },
                    {
                        routine: {
                            translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } },
                        }
                    },
                    { title: { ...insensitive } }
                ]
            })
        })
    },
    customQueries(input: RunSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            (input.completedTimeFrame !== undefined ? timeFrameToPrisma('timeCompleted', input.completedTimeFrame) : {}),
            (input.startedTimeFrame !== undefined ? timeFrameToPrisma('timeStarted', input.startedTimeFrame) : {}),
            (input.status !== undefined ? { status: input.status } : {}),
        ])
    },
})

export const runValidator = (): Validator<
    RunCreateInput,
    RunUpdateInput,
    Run,
    Prisma.runGetPayload<{ select: { [K in keyof Required<Prisma.runSelect>]: true } }>,
    RunPermission,
    Prisma.runSelect,
    Prisma.runWhereInput
> => ({
    validateMap: {
        __typename: 'Run',
        asdffasdf
    },
    permissionsSelect: (userId) => ({ 
        id: true, 
        ...permissionsSelectHelper([
            ['organization', 'Organization'],
            ['routineVersion', 'Routine'],
            ['user', 'User'],
        ], userId)
    }),
    permissionsFromSelect: (select, userId) => asdf as any,
    isPublic: (data) => data.isPrivate === false && ownOne<Prisma.runSelect>(data, [
        ['organization', 'Organization'],
        ['user', 'User'],
    ]),
    // profanityCheck(data: (RunCreateInput | RunUpdateInput)[]): void {
    //     validateProfanity(data.map((d: any) => d.title));
    // },

    // TODO if status passed in for update, make sure it's not the same 
    // as the current status, or an invalid transition (e.g. failed -> in progress)
})

/**
 * Handles run instances of routines
 */
export const runMutater = (prisma: PrismaType): Mutater<Run> => ({
    async shapeCreate(userId: string, data: RunCreateInput): Promise<Prisma.runUpsertArgs['create']> {
        // TODO - when scheduling added, don't assume that it is being started right away
        return {
            id: data.id,
            timeStarted: new Date(),
            routineVersionId: data.routineVersionId,
            status: RunStatus.InProgress,
            steps: await RunStepModel.mutate(prisma).relationshipBuilder(userId, data, true, 'step'),
            title: data.title,
            userId,
        }
    },
    async shapeUpdate(userId: string, data: RunUpdateInput): Promise<Prisma.runUpsertArgs['update']> {
        return {
            timeElapsed: data.timeElapsed ? { increment: data.timeElapsed } : undefined,
            completedComplexity: data.completedComplexity ? { increment: data.completedComplexity } : undefined,
            contextSwitches: data.contextSwitches ? { increment: data.contextSwitches } : undefined,
            steps: await RunStepModel.mutate(prisma).relationshipBuilder(userId, data, false),
            inputs: await RunInputModel.mutate(prisma).relationshipBuilder(userId, data, false),
        }
    },
    async cud(params: CUDInput<RunCreateInput, RunUpdateInput>): Promise<CUDResult<Run>> {
        return cudHelper({
            ...params,
            objectType: 'Run',
            prisma,
            yup: { yupCreate: runsCreate, yupUpdate: runsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                // Handle run start trigger for every run with status InProgress
                for (const c of created) {
                    if (c.status === RunStatus.InProgress) {
                        Trigger(prisma).runStart(c.title as string, c.id as string, params.userData.id, false);
                    }
                }
            },
            onUpdated: (updated, updateData) => {
                for (let i = 0; i < updated.length; i++) {
                    // Handle run start trigger for every run with status InProgress, 
                    // that previously had a status of Scheduled
                    if (updated[i].status === RunStatus.InProgress && updateData[i].hasOwnProperty('status')) {
                        Trigger(prisma).runStart(updated[i].title as string, updated[i].id as string, params.userData.id, false);
                    }
                    // Handle run complete trigger for every run with status Completed,
                    // that previously had a status of InProgress
                    if (updated[i].status === RunStatus.Completed && updateData[i].hasOwnProperty('status')) {
                        Trigger(prisma).runComplete(updated[i].title as string, updated[i].id as string, params.userData.id, false);
                    }
                    // Handle run fail trigger for every run with status Failed,
                    // that previously had a status of InProgress
                    if (updated[i].status === RunStatus.Failed && updateData[i].hasOwnProperty('status')) {
                        Trigger(prisma).runFail(updated[i].title as string, updated[i].id as string, params.userData.id, false);
                    }
                }
            }
        })
    },
    /**
     * Deletes all runs for a user, except if they are in progress
     */
    async deleteAll(userId: string): Promise<Count> {
        return prisma.run.deleteMany({
            where: {
                AND: [
                    { userId },
                    { NOT: { status: RunStatus.InProgress } }
                ]
            }
        });
    },
    /**
     * Marks a run as completed. Run does not have to exist, since this can be called on simple routines 
     * via the "Mark as Complete" button. We could create a new run every time a simple routine is viewed 
     * to get around this, but I'm not sure if that would be a good idea. Most of the time, I imagine users
     * will just be looking at the routine instead of using it.
     */
    async complete(userId: string, input: RunCompleteInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, runFormatter().relationshipMap);
        if (partial === undefined) throw new CustomError(CODE.ErrorUnknown, 'Invalid query.', { code: genErrorCode('0179') });
        let run: run | null;
        // Check if run is being created or updated
        if (input.exists) {
            // Find in database
            run = await prisma.run.findFirst({
                where: {
                    AND: [
                        { userId },
                        { id: input.id },
                    ]
                }
            })
            if (!run) throw new CustomError(CODE.NotFound, 'Run not found.', { code: genErrorCode('0180') });
            const { timeElapsed, contextSwitches, completedComplexity } = run;
            // Update object
            run = await prisma.run.update({
                where: { id: input.id },
                data: {
                    completedComplexity: completedComplexity + (input.completedComplexity ?? 0),
                    contextSwitches: contextSwitches + (input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0),
                    status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
                    timeCompleted: new Date(),
                    timeElapsed: (timeElapsed ?? 0) + (input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0),
                    steps: {
                        create: input.finalStepCreate ? {
                            order: input.finalStepCreate.order ?? 1,
                            title: input.finalStepCreate.title ?? '',
                            contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepCreate.timeElapsed,
                            status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
                        } as any : undefined,
                        update: input.finalStepUpdate ? {
                            id: input.finalStepUpdate.id,
                            contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepUpdate.timeElapsed,
                            status: input.finalStepUpdate.status ?? (input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed),
                        } as any : undefined,
                    }
                    //TODO
                    // inputs: {
                    //     create: input.finalInputCreate ? {
                    // }
                },
                ...selectHelper(partial)
            });
        } else {
            // Create new run
            run = await prisma.run.create({
                data: {
                    completedComplexity: input.completedComplexity ?? 0,
                    timeStarted: new Date(),
                    timeCompleted: new Date(),
                    timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
                    contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
                    routineVersionId: input.id,
                    status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                    title: input.title,
                    userId,
                    steps: {
                        create: input.finalStepCreate ? {
                            order: input.finalStepCreate.order ?? 1,
                            title: input.finalStepCreate.title ?? '',
                            contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepCreate.timeElapsed,
                            status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                        } as any : input.finalStepUpdate ? {
                            id: input.finalStepUpdate.id,
                            contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                            timeElapsed: input.finalStepUpdate.timeElapsed,
                            status: input.finalStepUpdate?.status ?? (input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed),
                        } : undefined,
                    }
                    //TODO inputs
                },
                ...selectHelper(partial)
            });
        }
        // Convert to GraphQL
        let converted: any = modelToGraphQL(run, partial);
        // Add supplemental fields
        converted = (await addSupplementalFields(prisma, userId, [converted], partial))[0];
        // Handle trigger
        if (input.wasSuccessful) await Trigger(prisma).runComplete(input.title, input.id, userId, false);
        else await Trigger(prisma).runFail(input.title, input.id, userId, false);
        // Return converted object
        return converted as Run;
    },
    /**
     * Cancels a run
     */
    async cancel(userId: string, input: RunCancelInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, runFormatter().relationshipMap);
        if (partial === undefined) throw new CustomError(CODE.ErrorUnknown, 'Invalid query.', { code: genErrorCode('0181') });
        // Find in database
        let object = await prisma.run.findFirst({
            where: {
                AND: [
                    { userId },
                    { id: input.id },
                ]
            }
        })
        if (!object) throw new CustomError(CODE.NotFound, 'Run not found.', { code: genErrorCode('0182') });
        // Update object
        const updated = await prisma.run.update({
            where: { id: input.id },
            data: {
                status: RunStatus.Cancelled,
            },
            ...selectHelper(partial)
        });
        // Convert to GraphQL
        let converted: any = modelToGraphQL(updated, partial);
        // Add supplemental fields
        converted = (await addSupplementalFields(prisma, userId, [converted], partial))[0];
        // Return converted object
        return converted as Run;
    },
})

export const RunModel = ({
    prismaObject: (prisma: PrismaType) => prisma.run,
    format: runFormatter(),
    mutate: runMutater,
    search: runSearcher(),
    type: 'Run' as GraphQLModelType,
    validate: runValidator(),
})