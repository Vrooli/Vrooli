import { MaxObjects, RunSortBy, runValidation } from "@local/shared";
import { RunStatus, RunStepStatus } from "@prisma/client";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { Trigger } from "../../events/trigger.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { RunFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ResourceVersionModelLogic, RunModelInfo, RunModelLogic } from "./types.js";

const __typename = "Run" as const;
export const RunModel: RunModelLogic = ({
    __typename,
    danger: {
        async anonymize(owner) {
            await DbProvider.get().run.updateMany({
                where: {
                    teamId: owner.__typename === "Team" ? BigInt(owner.id) : undefined,
                    userId: owner.__typename === "User" ? BigInt(owner.id) : undefined,
                    isPrivate: false,
                },
                data: {
                    teamId: null,
                    userId: null,
                },
            });
        },
        async deleteAll(owner) {
            const result = await DbProvider.get().run.deleteMany({
                where: {
                    teamId: owner.__typename === "Team" ? BigInt(owner.id) : undefined,
                    userId: owner.__typename === "User" ? BigInt(owner.id) : undefined,
                },
            });
            return result.count;
        },
    },
    dbTable: "run",
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
    format: RunFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    id: BigInt(data.id),
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches,
                    data: noNull(data.data),
                    embeddingNeedsUpdate: true,
                    isPrivate: data.isPrivate,
                    name: data.name,
                    status: noNull(data.status),
                    timeElapsed,
                    startedAt: data.startedAt,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    user: data.teamConnect ? undefined : { connect: { id: BigInt(rest.userData.id) } },
                    resourceVersion: await shapeHelper({ relation: "resourceVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ResourceVersion", parentRelationshipName: "runs", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runs", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, objectType: "RunStep", parentRelationshipName: "run", data, ...rest }),
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "runs", data, ...rest }),
                    io: await shapeHelper({ relation: "io", relTypes: ["Create"], isOneToOne: false, objectType: "RunIO", parentRelationshipName: "run", data, ...rest }),
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
                    data: noNull(data.data),
                    isPrivate: noNull(data.isPrivate),
                    status: noNull(data.status),
                    timeElapsed,
                    // TODO should have way of checking old status, so we don't reset startedAt/completedAt
                    startedAt: data.startedAt,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update", "Delete"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runs", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunStep", parentRelationshipName: "run", data, ...rest }),
                    io: await shapeHelper({ relation: "io", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunIO", parentRelationshipName: "run", data, ...rest }),
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
        yup: runValidation,
    },
    search: {
        defaultSort: RunSortBy.DateUpdatedDesc,
        sortBy: RunSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdTimeFrame: true,
            excludeIds: true,
            resourceVersionId: true,
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
                { resourceVersion: ModelMap.get<ResourceVersionModelLogic>("ResourceVersion").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            // Add fields needed for notifications when a run is started/completed
            dbFields: ["name"],
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                // Convert string IDs to BigInts for the query parameter
                const bigIntIds = ids.map(id => BigInt(id));
                // Find the step with the highest "completedAt" Date for each run
                const recentSteps = await DbProvider.get().$queryRaw`
                    SELECT DISTINCT ON ("runId")
                    "runId",
                    step
                    FROM run_step
                    WHERE "runId" = ANY(${bigIntIds})
                    AND "completedAt" IS NOT NULL
                    AND status = 'Completed'
                    ORDER BY "runId", "completedAt" DESC
                ` as { runId: bigint, step: RunStepStatus }[];
                const stepMap = new Map(recentSteps.map(step => [step.runId.toString(), step.step]));
                const lastSteps = ids.map(id => stepMap.get(id) || null);
                return {
                    lastStep: lastSteps,
                    you: {
                        ...(await getSingleTypePermissions<RunModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
                oneIsPublic<RunModelInfo["DbSelect"]>([
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
