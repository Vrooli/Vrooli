import { Count, MaxObjects, RunRoutine, RunRoutineCancelInput, RunRoutineCompleteInput, RunRoutineSortBy, runRoutineValidation } from "@local/shared";
import { RunStatus, run_routine } from "@prisma/client";
import { ModelMap } from ".";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { noNull } from "../../builders/noNull";
import { selectHelper } from "../../builders/selectHelper";
import { shapeHelper } from "../../builders/shapeHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { GraphQLInfo } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { SessionUserToken } from "../../types";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { RunRoutineFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { RoutineVersionModelLogic, RunRoutineModelInfo, RunRoutineModelLogic, TeamModelLogic } from "./types";

const __typename = "RunRoutine" as const;
export const RunRoutineModel: RunRoutineModelLogic = ({
    __typename,
    danger: {
        /**
         * Anonymizes all public runs associated with a user or team
         */
        async anonymize(owner: { __typename: "Team" | "User", id: string }): Promise<void> {
            await prismaInstance.run_routine.updateMany({
                where: {
                    teamId: owner.__typename === "Team" ? owner.id : undefined,
                    userId: owner.__typename === "User" ? owner.id : undefined,
                    isPrivate: false,
                },
                data: {
                    teamId: null,
                    userId: null,
                },
            });
        },
        /**
         * Deletes all runs associated with a user or team
         */
        async deleteAll(owner: { __typename: "Team" | "User", id: string }): Promise<Count> {
            return prismaInstance.run_routine.deleteMany({
                where: {
                    teamId: owner.__typename === "Team" ? owner.id : undefined,
                    userId: owner.__typename === "User" ? owner.id : undefined,
                },
            }).then(({ count }) => ({ __typename: "Count" as const, count })) as any;
        },
    },
    dbTable: "run_routine",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
        embed: {
            select: () => ({ id: true, embeddingNeedsUpdate: true, name: true }),
            get: ({ name }, languages) => {
                return getEmbeddableString({ name }, languages[0]);
            },
        },
    }),
    format: RunRoutineFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches: noNull(data.contextSwitches),
                    embeddingNeedsUpdate: true,
                    isPrivate: data.isPrivate,
                    name: data.name,
                    status: noNull(data.status),
                    startedAt: data.status === RunStatus.InProgress ? new Date() : undefined,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    user: data.teamConnect ? undefined : { connect: { id: rest.userData.id } },
                    routineVersion: await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "runRoutines", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runRoutines", data, ...rest }),
                    runProject: await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, objectType: "RunProject", parentRelationshipName: "runRoutines", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "runRoutines", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineInput", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches: noNull(data.contextSwitches),
                    isPrivate: noNull(data.isPrivate),
                    status: noNull(data.status),
                    timeElapsed: noNull(data.timeElapsed),
                    // TODO should have way of checking old status, so we don't reset startedAt/completedAt
                    startedAt: data.status === RunStatus.InProgress ? new Date() : undefined,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runRoutines", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineInput", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: ({ createInputs, createdIds, updatedIds, updateInputs, userData }) => {
                // Handle run start trigger for every run with status InProgress
                for (const { id, status } of createInputs) {
                    if (status === RunStatus.InProgress) {
                        Trigger(userData.languages).runRoutineStart(id, userData.id, false);
                    }
                }
                for (let i = 0; i < updatedIds.length; i++) {
                    const { id, status } = updateInputs[i];
                    if (!status) continue;
                    // Handle run start trigger for every run with status InProgress, 
                    // that previously had a status of Scheduled
                    if (status === RunStatus.InProgress && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runRoutineStart(id, userData.id, false);
                    }
                    // Handle run complete trigger for every run with status Completed,
                    // that previously had a status of InProgress
                    if (status === RunStatus.Completed && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runRoutineComplete(id, userData.id, false);
                    }
                    // Handle run fail trigger for every run with status Failed,
                    // that previously had a status of InProgress
                    if (status === RunStatus.Failed && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runRoutineFail(id, userData.id, false);
                    }
                }
            },
        },
        yup: runRoutineValidation,
    },
    run: {
        /**
         * Marks a run as completed. Run does not have to exist, since this can be called on simple routines 
         * via the "Mark as Complete" button. We could create a new run every time a simple routine is viewed 
         * to get around this, but I'm not sure if that would be a good idea. Most of the time, I imagine users
         * will just be looking at the routine instead of using it.
         */
        async complete(userData: SessionUserToken, input: RunRoutineCompleteInput, info: GraphQLInfo): Promise<RunRoutine> {
            // Convert info to partial
            const partial = toPartialGqlInfo(info, ModelMap.get<RunRoutineModelLogic>("RunRoutine").format.gqlRelMap, userData.languages, true);
            let run: run_routine | null;
            // Check if run is being created or updated
            if (input.exists) {
                // Find in database
                run = await prismaInstance.run_routine.findFirst({
                    where: {
                        AND: [
                            { userId: userData.id },
                            { id: input.id },
                        ],
                    },
                });
                if (!run) throw new CustomError("0180", "NotFound", userData.languages);
                const { timeElapsed, contextSwitches, completedComplexity } = run;
                // Update object
                run = await prismaInstance.run_routine.update({
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
                                name: input.finalStepCreate.name ?? "",
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
                        },
                        //TODO
                        // inputs: {
                        //     create: input.finalInputCreate ? {
                        // }
                    },
                    ...selectHelper(partial),
                });
            } else {
                // Create new run
                run = await prismaInstance.run_routine.create({
                    data: {
                        completedComplexity: input.completedComplexity ?? 0,
                        startedAt: new Date(),
                        completedAt: new Date(),
                        timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
                        contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
                        routineVersionId: input.id,
                        status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                        name: input.name ?? "",
                        userId: userData.id,
                        steps: {
                            create: input.finalStepCreate ? {
                                order: input.finalStepCreate.order ?? 1,
                                name: input.finalStepCreate.name ?? "",
                                contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                                timeElapsed: input.finalStepCreate.timeElapsed,
                                status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                            } as any : input.finalStepUpdate ? {
                                id: input.finalStepUpdate.id,
                                contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                                timeElapsed: input.finalStepUpdate.timeElapsed,
                                status: input.finalStepUpdate?.status ?? (input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed),
                            } : undefined,
                        },
                        //TODO inputs
                    },
                    ...selectHelper(partial),
                });
            }
            // Convert to GraphQL
            let converted: any = modelToGql(run, partial);
            // Add supplemental fields
            converted = (await addSupplementalFields(userData, [converted], partial))[0];
            // Handle trigger
            if (input.wasSuccessful) await Trigger(userData.languages).runRoutineComplete(input.id, userData.id, false);
            else await Trigger(userData.languages).runRoutineFail(input.id, userData.id, false);
            // Return converted object
            return converted as RunRoutine;
        },
        /**
         * Cancels a run
         */
        async cancel(userData: SessionUserToken, input: RunRoutineCancelInput, info: GraphQLInfo): Promise<RunRoutine> {
            // Convert info to partial
            const partial = toPartialGqlInfo(info, ModelMap.get<RunRoutineModelLogic>("RunRoutine").format.gqlRelMap, userData.languages, true);
            // Find in database
            const object = await prismaInstance.run_routine.findFirst({
                where: {
                    AND: [
                        { userId: userData.id },
                        { id: input.id },
                    ],
                },
            });
            if (!object) throw new CustomError("0182", "NotFound", userData.languages);
            // Update object
            const updated = await prismaInstance.run_routine.update({
                where: { id: input.id },
                data: {
                    status: RunStatus.Cancelled,
                },
                ...selectHelper(partial),
            });
            // Convert to GraphQL
            let converted: any = modelToGql(updated, partial);
            // Add supplemental fields
            converted = (await addSupplementalFields(userData, [converted], partial))[0];
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
            routineVersionId: true,
            scheduleEndTimeFrame: true,
            scheduleStartTimeFrame: true,
            startedTimeFrame: true,
            status: true,
            statuses: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            // Add fields needed for notifications when a run is started/completed
            dbFields: ["name"],
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            routineVersion: "RoutineVersion",
            team: "Team",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Team: data?.team,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: (data, ...rest) =>
            data.isPrivate === false &&
            (
                (data.user === null && data.team === null) ||
                oneIsPublic<RunRoutineModelInfo["PrismaSelect"]>([
                    ["team", "Team"],
                    ["user", "User"],
                ], data, ...rest)
            ),
        profanityFields: ["name"],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { user: { id: userId } },
                ],
            }),
        },
    }),
});
