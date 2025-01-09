import { MaxObjects, RunRoutineSortBy, runRoutineValidation } from "@local/shared";
import { RunStatus, RunStepStatus } from "@prisma/client";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { Trigger } from "../../events/trigger";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { RunRoutineFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { RoutineVersionModelLogic, RunRoutineModelInfo, RunRoutineModelLogic } from "./types";

const __typename = "RunRoutine" as const;
export const RunRoutineModel: RunRoutineModelLogic = ({
    __typename,
    danger: {
        async anonymize(owner) {
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
        async deleteAll(owner) {
            const result = await prismaInstance.run_routine.deleteMany({
                where: {
                    teamId: owner.__typename === "Team" ? owner.id : undefined,
                    userId: owner.__typename === "User" ? owner.id : undefined,
                },
            });
            return result.count;
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
                return getEmbeddableString({ name }, languages?.[0]);
            },
        },
    }),
    format: RunRoutineFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    id: data.id,
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches,
                    embeddingNeedsUpdate: true,
                    isPrivate: data.isPrivate,
                    name: data.name,
                    status: noNull(data.status),
                    timeElapsed,
                    startedAt: data.status === RunStatus.InProgress ? new Date() : undefined,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    user: data.teamConnect ? undefined : { connect: { id: rest.userData.id } },
                    routineVersion: await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "runRoutines", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runRoutines", data, ...rest }),
                    runProject: await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, objectType: "RunProject", parentRelationshipName: "runRoutines", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "runRoutines", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineInput", parentRelationshipName: "runRoutine", data, ...rest }),
                    outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineOutput", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches,
                    isPrivate: noNull(data.isPrivate),
                    status: noNull(data.status),
                    timeElapsed,
                    // TODO should have way of checking old status, so we don't reset startedAt/completedAt
                    startedAt: data.status === RunStatus.InProgress ? new Date() : undefined,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runRoutines", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                    inputs: await shapeHelper({ relation: "inputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineInput", parentRelationshipName: "runRoutine", data, ...rest }),
                    outputs: await shapeHelper({ relation: "outputs", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineOutput", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: ({ createInputs, updatedIds, updateInputs, userData }) => {
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                // Find the step with the highest "completedAt" Date for each run
                const recentSteps = await prismaInstance.$queryRaw`
                    SELECT DISTINCT ON ("runRoutineId")
                    "runRoutineId",
                    step
                    FROM run_routine_step
                    WHERE "runRoutineId" = ANY(${ids}::uuid[])
                    AND "completedAt" IS NOT NULL
                    AND status = 'Completed'
                    ORDER BY "runRoutineId", "completedAt" DESC
                ` as { runRoutineId: string, step: RunStepStatus }[];
                const stepMap = new Map(recentSteps.map(step => [step.runRoutineId, step.step]));
                const lastSteps = ids.map(id => stepMap.get(id) || null);
                return {
                    lastStep: lastSteps,
                    you: {
                        ...(await getSingleTypePermissions<RunRoutineModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
                oneIsPublic<RunRoutineModelInfo["DbSelect"]>([
                    ["team", "Team"],
                    ["user", "User"],
                ], data, ...rest)
            ),
        profanityFields: ["name"],
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { team: useVisibility("Team", "Own", data) },
                        { user: useVisibility("User", "Own", data) },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { team: useVisibility("Team", "OwnOrPublic", data) },
                        { user: useVisibility("User", "OwnOrPublic", data) },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isPrivate: true,
                    OR: [
                        { team: useVisibility("Team", "Own", data) },
                        { user: useVisibility("User", "Own", data) },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isPrivate: false,
                    OR: [
                        { team: useVisibility("Team", "Own", data) },
                        { user: useVisibility("User", "Own", data) },
                    ],
                };
            },
            public: function getPublic() {
                return {
                    isPrivate: false,
                };
            },
        },
    }),
});
