import { runsCreate, runsUpdate } from "@shared/validation";
import { RunSortBy } from "@shared/consts";
import { addSupplementalFields, modelToGraphQL, selectHelper, timeFrameToPrisma, toPartialGraphQLInfo, combineQueries, permissionsSelectHelper } from "./builder";
import { RunStepModel } from "./runStep";
import { Prisma, run_routine, RunStatus } from "@prisma/client";
import { RunInputModel } from "./runInput";
import { CustomError, Trigger } from "../events";
import { Run, RunSearchInput, RunCreateInput, RunUpdateInput, RunPermission, Count, RunCompleteInput, RunCancelInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, GraphQLInfo, Validator, Mutater } from "./types";
import { oneIsPublic } from "./utils";
import { OrganizationModel } from "./organization";

const formatter = (): Formatter<Run, ''> => ({
    relationshipMap: {
        __typename: 'RunRoutine',
        routine: 'Routine',
        steps: 'RunStep',
        inputs: 'RunInput',
        user: 'User',
    },
    supplemental: {
        // Add fields needed for notifications when a run is started/completed
        dbFields: ['title'],
        graphqlFields: [],
        toGraphQL: () => [],
    },
})

const searcher = (): Searcher<
    RunSearchInput,
    RunSortBy,
    Prisma.run_routineOrderByWithRelationInput,
    Prisma.run_routineWhereInput
> => ({
    defaultSort: RunSortBy.DateUpdatedDesc,
    sortMap: {
        DateStartedAsc: { timeStarted: 'asc' },
        DateStartedDesc: { timeStarted: 'desc' },
        DateCompletedAsc: { timeCompleted: 'asc' },
        DateCompletedDesc: { timeCompleted: 'desc' },
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
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
    }),
    customQueries(input) {
        return combineQueries([
            (input.routineId !== undefined ? { routines: { some: { id: input.routineId } } } : {}),
            (input.completedTimeFrame !== undefined ? timeFrameToPrisma('timeCompleted', input.completedTimeFrame) : {}),
            (input.startedTimeFrame !== undefined ? timeFrameToPrisma('timeStarted', input.startedTimeFrame) : {}),
            (input.status !== undefined ? { status: input.status } : {}),
        ])
    },
})

const validator = (): Validator<
    RunCreateInput,
    RunUpdateInput,
    Run,
    Prisma.run_routineGetPayload<{ select: { [K in keyof Required<Prisma.run_routineSelect>]: true } }>,
    RunPermission,
    Prisma.run_routineSelect,
    Prisma.run_routineWhereInput
> => ({
    validateMap: {
        __typename: 'RunRoutine',
        routineVersion: 'Routine',
        organization: 'Organization',
        user: 'User',
    },
    permissionsSelect: (...params) => ({
        id: true,
        isPrivate: true,
        ...permissionsSelectHelper([
            ['organization', 'Organization'],
            ['routineVersion', 'Routine'],
            ['user', 'User'],
        ], ...params)
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        ['canDelete', async () => isAdmin && !isDeleted],
        ['canEdit', async () => isAdmin && !isDeleted],
        ['canView', async () => !isDeleted && (isAdmin || isPublic)],
    ]),
    owner: (data) => ({
        Organization: data.organization,
        User: data.user,
    }),
    isDeleted: () => false,
    isPublic: (data, languages,) => data.isPrivate === false && oneIsPublic<Prisma.run_routineSelect>(data, [
        ['organization', 'Organization'],
        ['user', 'User'],
    ], languages),
    ownerOrMemberWhere: (userId) => ({
        OR: [
            OrganizationModel.query.hasRoleInOrganizationQuery(userId),
            { user: { id: userId } }
        ]
    }),
    // profanityCheck(data: (RunCreateInput | RunUpdateInput)[]): void {
    //     validateProfanity(data.map((d: any) => d.title));
    // },

    // TODO if status passed in for update, make sure it's not the same 
    // as the current status, or an invalid transition (e.g. failed -> in progress)
})

const runner = () => ({
    /**
     * Marks a run as completed. Run does not have to exist, since this can be called on simple routines 
     * via the "Mark as Complete" button. We could create a new run every time a simple routine is viewed 
     * to get around this, but I'm not sure if that would be a good idea. Most of the time, I imagine users
     * will just be looking at the routine instead of using it.
     */
    async complete(prisma: PrismaType, userData: SessionUser, input: RunCompleteInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, formatter().relationshipMap);
        if (partial === undefined) throw new CustomError('0179', 'ErrorUnknown', userData.languages);
        let run: run_routine | null;
        // Check if run is being created or updated
        if (input.exists) {
            // Find in database
            run = await prisma.run_routine.findFirst({
                where: {
                    AND: [
                        { userId: userData.id },
                        { id: input.id },
                    ]
                }
            })
            if (!run) throw new CustomError('0180', 'NotFound', userData.languages);
            const { timeElapsed, contextSwitches, completedComplexity } = run;
            // Update object
            run = await prisma.run_routine.update({
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
            run = await prisma.run_routine.create({
                data: {
                    completedComplexity: input.completedComplexity ?? 0,
                    timeStarted: new Date(),
                    timeCompleted: new Date(),
                    timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
                    contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
                    routineVersionId: input.id,
                    status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                    title: input.title,
                    userId: userData.id,
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
        converted = (await addSupplementalFields(prisma, userData, [converted], partial))[0];
        // Handle trigger
        if (input.wasSuccessful) await Trigger(prisma, userData.languages).runComplete(input.title, input.id, userData.id, false);
        else await Trigger(prisma, userData.languages).runFail(input.title, input.id, userData.id, false);
        // Return converted object
        return converted as Run;
    },
    /**
     * Cancels a run
     */
    async cancel(prisma: PrismaType, userData: SessionUser, input: RunCancelInput, info: GraphQLInfo): Promise<Run> {
        // Convert info to partial
        const partial = toPartialGraphQLInfo(info, formatter().relationshipMap);
        if (partial === undefined) throw new CustomError('181', 'ErrorUnknown', userData.languages);
        // Find in database
        let object = await prisma.run_routine.findFirst({
            where: {
                AND: [
                    { userId: userData.id },
                    { id: input.id },
                ]
            }
        })
        if (!object) throw new CustomError('0182', 'NotFound', userData.languages);
        // Update object
        const updated = await prisma.run_routine.update({
            where: { id: input.id },
            data: {
                status: RunStatus.Cancelled,
            },
            ...selectHelper(partial)
        });
        // Convert to GraphQL
        let converted: any = modelToGraphQL(updated, partial);
        // Add supplemental fields
        converted = (await addSupplementalFields(prisma, userData, [converted], partial))[0];
        // Return converted object
        return converted as Run;
    },
})

const mutater = (): Mutater<
    Run,
    { graphql: RunCreateInput, db: Prisma.run_routineUpsertArgs['create'] },
    { graphql: RunUpdateInput, db: Prisma.run_routineUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            // TODO - when scheduling added, don't assume that it is being started right away
            return {
                id: data.id,
                timeStarted: new Date(),
                routineVersionId: data.routineVersionId,
                status: RunStatus.InProgress,
                steps: await RunStepModel.mutate.relationshipBuilder!(prisma, userData, data, true, 'step'),
                title: data.title,
                userId: userData.id,
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                timeElapsed: data.timeElapsed ? { increment: data.timeElapsed } : undefined,
                completedComplexity: data.completedComplexity ? { increment: data.completedComplexity } : undefined,
                contextSwitches: data.contextSwitches ? { increment: data.contextSwitches } : undefined,
                steps: await RunStepModel.mutate.relationshipBuilder!(prisma, userData, data, false),
                inputs: await RunInputModel.mutate.relationshipBuilder!(prisma, userData, data, false),
            }
        }
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            // Handle run start trigger for every run with status InProgress
            for (const c of created) {
                if (c.status === RunStatus.InProgress) {
                    Trigger(prisma, userData.languages).runStart(c.title as string, c.id as string, userData.id, false);
                }
            }
        },
        onUpdated: ({ prisma, updated, updateInput, userData }) => {
            for (let i = 0; i < updated.length; i++) {
                // Handle run start trigger for every run with status InProgress, 
                // that previously had a status of Scheduled
                if (updated[i].status === RunStatus.InProgress && updateInput[i].hasOwnProperty('status')) {
                    Trigger(prisma, userData.languages).runStart(updated[i].title as string, updated[i].id as string, userData.id, false);
                }
                // Handle run complete trigger for every run with status Completed,
                // that previously had a status of InProgress
                if (updated[i].status === RunStatus.Completed && updateInput[i].hasOwnProperty('status')) {
                    Trigger(prisma, userData.languages).runComplete(updated[i].title as string, updated[i].id as string, userData.id, false);
                }
                // Handle run fail trigger for every run with status Failed,
                // that previously had a status of InProgress
                if (updated[i].status === RunStatus.Failed && updateInput[i].hasOwnProperty('status')) {
                    Trigger(prisma, userData.languages).runFail(updated[i].title as string, updated[i].id as string, userData.id, false);
                }
            }
        },
    },
    yup: { create: runsCreate, update: runsUpdate },
})

const danger = () => ({
    /**
     * Anonymizes all public runs associated with a user or organization
     */
    async anonymize(prisma: PrismaType, owner: { __typename: 'User' | 'Organization', id: string }): Promise<void> {
        await prisma.run_routine.updateMany({
            where: {
                userId: owner.__typename === 'User' ? owner.id : undefined,
                organizationId: owner.__typename === 'Organization' ? owner.id : undefined,
                isPrivate: false,
            },
            data: {
                userId: null,
                organizationId: null,
            }
        });
    },
    /**
     * Deletes all runs associated with a user or organization
     */
    async deleteAll(prisma: PrismaType, owner: { __typename: 'User' | 'Organization', id: string }): Promise<Count> {
        return prisma.run_routine.deleteMany({
            where: {
                userId: owner.__typename === 'User' ? owner.id : undefined,
                organizationId: owner.__typename === 'Organization' ? owner.id : undefined,
            }
        });
    }
})

export const RunModel = ({
    danger: danger(),
    delegate: (prisma: PrismaType) => prisma.run_routine,
    format: formatter(),
    mutate: mutater(),
    run: runner(),
    search: searcher(),
    type: 'Run' as GraphQLModelType,
    validate: validator(),
})