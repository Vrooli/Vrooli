import { Prisma, run_routine, RunStatus } from "@prisma/client";
import { CustomError, Trigger } from "../events";
import { RunRoutine, RunRoutineSearchInput, RunRoutineCreateInput, RunRoutineUpdateInput, RunRoutinePermission, Count, RunRoutineCompleteInput, RunRoutineCancelInput, SessionUser, RunRoutineSortBy } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { OrganizationModel } from "./organization";
import { addSupplementalFields, modelToGraphQL, permissionsSelectHelper, selectHelper, toPartialGraphQLInfo } from "../builders";
import { oneIsPublic } from "../utils";
import { GraphQLInfo, SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunRoutineCreateInput,
    GqlUpdate: RunRoutineUpdateInput,
    GqlModel: RunRoutine,
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    GqlPermission: RunRoutinePermission,
    PrismaCreate: Prisma.run_routineUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routineUpsertArgs['update'],
    PrismaModel: Prisma.run_routineGetPayload<SelectWrap<Prisma.run_routineSelect>>,
    PrismaSelect: Prisma.run_routineSelect,
    PrismaWhere: Prisma.run_routineWhereInput,
}

const __typename = 'RunRoutine' as const;
const suppFields = [] as const;
export const RunRoutineModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    danger: {
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
    },
    delegate: (prisma: PrismaType) => prisma.run_routine,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            inputs: 'RunRoutineInput',
            organization: 'Organization',
            routineVersion: 'RoutineVersion',
            runProject: 'RunProject',
            runRoutineSchedule: 'RunRoutineSchedule',
            steps: 'RunRoutineStep',
            user: 'User',
        },
        prismaRelMap: {
            __typename,
            inputs: 'RunRoutineInput',
            organization: 'Organization',
            routineVersion: 'Routine',
            runProject: 'RunProject',
            runRoutineSchedule: 'RunRoutineSchedule',
            steps: 'RunRoutineStep',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            // Add fields needed for notifications when a run is started/completed
            dbFields: ['name'],
            graphqlFields: suppFields,
            toGraphQL: () => [],
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => {
                // TODO - when scheduling added, don't assume that it is being started right away
                return {
                    // id: data.id,
                    // startedAt: new Date(),
                    // routineVersionId: data.routineVersionId,
                    // status: RunStatus.InProgress,
                    // steps: await relBuilderHelper({ data, isAdd: true, isOneToOne: false, isRequired: false, relationshipName: 'step', objectType: 'RunRoutineStep', prisma, userData }),
                    // name: data.name,
                    // userId: userData.id,
                } as any
            },
            update: async ({ data, prisma, userData }) => {
                return {
                    // timeElapsed: data.timeElapsed ? { increment: data.timeElapsed } : undefined,
                    // completedComplexity: data.completedComplexity ? { increment: data.completedComplexity } : undefined,
                    // contextSwitches: data.contextSwitches ? { increment: data.contextSwitches } : undefined,
                    // steps: await relBuilderHelper({ data, isAdd: false, isOneToOne: false, isRequired: false, relationshipName: 'step', objectType: 'RunRoutineStep', prisma, userData }),
                    // inputs: await relBuilderHelper({ data, isAdd: false, isOneToOne: false, isRequired: false, relationshipName: 'inputs', objectType: 'RunRoutineInput', prisma, userData }),
                } as any;
            }
        },
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                // Handle run start trigger for every run with status InProgress
                for (const c of created) {
                    if (c.status === RunStatus.InProgress) {
                        Trigger(prisma, userData.languages).runRoutineStart(c.id, userData.id, false);
                    }
                }
            },
            onUpdated: ({ prisma, updated, updateInput, userData }) => {
                for (let i = 0; i < updated.length; i++) {
                    // Handle run start trigger for every run with status InProgress, 
                    // that previously had a status of Scheduled
                    if (updated[i].status === RunStatus.InProgress && updateInput[i].hasOwnProperty('status')) {
                        Trigger(prisma, userData.languages).runRoutineStart(updated[i].id, userData.id, false);
                    }
                    // Handle run complete trigger for every run with status Completed,
                    // that previously had a status of InProgress
                    if (updated[i].status === RunStatus.Completed && updateInput[i].hasOwnProperty('status')) {
                        Trigger(prisma, userData.languages).runRoutineComplete(updated[i].id, userData.id, false);
                    }
                    // Handle run fail trigger for every run with status Failed,
                    // that previously had a status of InProgress
                    if (updated[i].status === RunStatus.Failed && updateInput[i].hasOwnProperty('status')) {
                        Trigger(prisma, userData.languages).runRoutineFail(updated[i].id, userData.id, false);
                    }
                }
            },
        },
        yup: {} as any,
    },
    run: {
        /**
         * Marks a run as completed. Run does not have to exist, since this can be called on simple routines 
         * via the "Mark as Complete" button. We could create a new run every time a simple routine is viewed 
         * to get around this, but I'm not sure if that would be a good idea. Most of the time, I imagine users
         * will just be looking at the routine instead of using it.
         */
        async complete(prisma: PrismaType, userData: SessionUser, input: RunRoutineCompleteInput, info: GraphQLInfo): Promise<RunRoutine> {
            // Convert info to partial
            const partial = toPartialGraphQLInfo(info, RunRoutineModel.format.gqlRelMap, userData.languages, true);
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
                        completedAt: new Date(),
                        timeElapsed: (timeElapsed ?? 0) + (input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0),
                        steps: {
                            create: input.finalStepCreate ? {
                                order: input.finalStepCreate.order ?? 1,
                                name: input.finalStepCreate.name ?? '',
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
                        startedAt: new Date(),
                        completedAt: new Date(),
                        timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
                        contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
                        routineVersionId: input.id,
                        status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                        name: input.name ?? '',
                        userId: userData.id,
                        steps: {
                            create: input.finalStepCreate ? {
                                order: input.finalStepCreate.order ?? 1,
                                name: input.finalStepCreate.name ?? '',
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
            if (input.wasSuccessful) await Trigger(prisma, userData.languages).runRoutineComplete(input.id, userData.id, false);
            else await Trigger(prisma, userData.languages).runRoutineFail(input.id, userData.id, false);
            // Return converted object
            return converted as RunRoutine;
        },
        /**
         * Cancels a run
         */
        async cancel(prisma: PrismaType, userData: SessionUser, input: RunRoutineCancelInput, info: GraphQLInfo): Promise<RunRoutine> {
            // Convert info to partial
            const partial = toPartialGraphQLInfo(info, RunRoutineModel.format.gqlRelMap, userData.languages, true);
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
            return converted as RunRoutine;
        },
    },
    search: {
        defaultSort: RunRoutineSortBy.DateUpdatedDesc,
        sortBy: RunRoutineSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdTimeFrame: true,
            excludeIds: true,
            routineId: true,
            startedTimeFrame: true,
            status: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'nameWrapped',
                { routineVersion: RunRoutineModel.search!.searchStringQuery() },
            ]
        })
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: 5000,
            Organization: 50000,
        },
        permissionsSelect: (...params) => ({
            id: true,
            isPrivate: true,
            ...permissionsSelectHelper({
                organization: 'Organization',
                routineVersion: 'Routine',
                user: 'User',
            }, ...params)
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            canDelete: async () => isAdmin && !isDeleted,
            canEdit: async () => isAdmin && !isDeleted,
            canView: async () => !isDeleted && (isAdmin || isPublic),
        }),
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages,) => data.isPrivate === false && oneIsPublic<Prisma.run_routineSelect>(data, [
            ['organization', 'Organization'],
            ['user', 'User'],
        ], languages),
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
        // profanityCheck(data: (RunCreateInput | RunUpdateInput)[]): void {
        //     validateProfanity(data.map((d: any) => d.name));
        // },

        // TODO if status passed in for update, make sure it's not the same 
        // as the current status, or an invalid transition (e.g. failed -> in progress)
    },
})