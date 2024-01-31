import { MaxObjects, RunProjectSortBy, runProjectValidation, RunStatus } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { RunProjectFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { OrganizationModelLogic, ProjectVersionModelLogic, RunProjectModelInfo, RunProjectModelLogic } from "./types";

const __typename = "RunProject" as const;
export const RunProjectModel: RunProjectModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.run_project,
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
    format: RunProjectFormat,
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
                    user: data.organizationConnect ? undefined : { connect: { id: rest.userData.id } },
                    projectVersion: await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "runProjects", data, ...rest }),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest }),
                    organization: await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, objectType: "Organization", parentRelationshipName: "runProjects", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches: noNull(data.contextSwitches),
                    isPrivate: noNull(data.isPrivate),
                    status: noNull(data.status),
                    timeElapsed: noNull(data.timeElapsed),
                    // TODO should have way to check previous status, so we don't overwrite startedAt/completedAt
                    startedAt: data.status === RunStatus.InProgress ? new Date() : undefined,
                    completedAt: data.status === RunStatus.Completed ? new Date() : undefined,
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest }),
                    steps: await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest }),
                };
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
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
            organization: "Organization",
            projectVersion: "ProjectVersion",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data?.organization,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: (data, ...rest) => data.isPrivate === false && oneIsPublic<RunProjectModelInfo["PrismaSelect"]>([
            ["organization", "Organization"],
            ["user", "User"],
        ], data, ...rest),
        profanityFields: ["name"],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
