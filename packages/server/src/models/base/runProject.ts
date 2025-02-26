import { MaxObjects, RunProjectSortBy, runProjectValidation, RunStatus } from "@local/shared";
import { RunStepStatus } from "@prisma/client";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { Trigger } from "../../events/trigger.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { RunProjectFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ProjectVersionModelLogic, RunProjectModelInfo, RunProjectModelLogic } from "./types.js";

const __typename = "RunProject" as const;
export const RunProjectModel: RunProjectModelLogic = ({
    __typename,
    danger: {
        async anonymize(owner) {
            await DbProvider.get().run_project.updateMany({
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
            const result = await DbProvider.get().run_project.deleteMany({
                where: {
                    teamId: owner.__typename === "Team" ? owner.id : undefined,
                    userId: owner.__typename === "User" ? owner.id : undefined,
                },
            });
            return result.count;
        },
    },
    dbTable: "run_project",
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
    format: RunProjectFormat,
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
                    data: noNull(data.data),
                    embeddingNeedsUpdate: true,
                    isPrivate: data.isPrivate,
                    name: data.name,
                    status: noNull(data.status),
                    timeElapsed,
                    startedAt: data.startedAt,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    user: data.teamConnect ? undefined : { connect: { id: rest.userData.id } },
                    projectVersion: await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "runProjects", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "runProjects", data, ...rest }),
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
                    // TODO should have way to check previous status, so we don't overwrite startedAt/completedAt
                    startedAt: data.startedAt,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update", "Delete"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: ({ createInputs, updatedIds, updateInputs, userData }) => {
                // Handle run start trigger for every run with status InProgress
                for (const { id, status } of createInputs) {
                    if (status === RunStatus.InProgress) {
                        Trigger(userData.languages).runProjectStart(id, userData.id, false);
                    }
                }
                for (let i = 0; i < updatedIds.length; i++) {
                    const { id, status } = updateInputs[i];
                    if (!status) continue;
                    // Handle run start trigger for every run with status InProgress, 
                    // that previously had a status of Scheduled
                    if (status === RunStatus.InProgress && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runProjectStart(id, userData.id, false);
                    }
                    // Handle run complete trigger for every run with status Completed,
                    // that previously had a status of InProgress
                    if (status === RunStatus.Completed && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runProjectComplete(id, userData.id, false);
                    }
                    // Handle run fail trigger for every run with status Failed,
                    // that previously had a status of InProgress
                    if (status === RunStatus.Failed && Object.prototype.hasOwnProperty.call(updateInputs[i], "status")) {
                        Trigger(userData.languages).runProjectFail(id, userData.id, false);
                    }
                }
            },
        },
        yup: runProjectValidation,
    },
    search: {
        defaultSort: RunProjectSortBy.DateUpdatedDesc,
        sortBy: RunProjectSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdTimeFrame: true,
            excludeIds: true,
            projectVersionId: true,
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
                { projectVersion: ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                // Find the step with the highest "completedAt" Date for each run
                const recentSteps = await DbProvider.get().$queryRaw`
                    SELECT DISTINCT ON ("runProjectId")
                    "runProjectId",
                    step
                    FROM run_project_step
                    WHERE "runProjectId" = ANY(${ids}::uuid[])
                    AND "completedAt" IS NOT NULL
                    AND status = 'Completed'
                    ORDER BY "runProjectId", "completedAt" DESC
                ` as { runProjectId: string, step: RunStepStatus }[];
                const stepMap = new Map(recentSteps.map(step => [step.runProjectId, step.step]));
                const lastSteps = ids.map(id => stepMap.get(id) || null);
                return {
                    lastStep: lastSteps,
                    you: {
                        ...(await getSingleTypePermissions<RunProjectModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
            projectVersion: "ProjectVersion",
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
                oneIsPublic<RunProjectModelInfo["DbSelect"]>([
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
