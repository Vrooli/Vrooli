import { MaxObjects, RunProjectSortBy, runProjectValidation, RunStatus } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { RunProjectFormat } from "../formats";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { ProjectVersionModel } from "./projectVersion";
import { RunProjectModelLogic } from "./types";

const __typename = "RunProject" as const;
const suppFields = ["you"] as const;
export const RunProjectModel: ModelLogic<RunProjectModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.run_project,
    display: {
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
    },
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
                    ...(data.status === RunStatus.InProgress ? { startedAt: new Date() } : {}),
                    ...(data.status === RunStatus.Completed ? { completedAt: new Date() } : {}),
                    ...(data.organizationConnect ? {} : { user: { connect: { id: rest.userData.id } } }),
                    ...(await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "ProjectVersion", parentRelationshipName: "runProjects", data, ...rest })),
                    ...(await shapeHelper({ relation: "schedule", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest })),
                    ...(await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Organization", parentRelationshipName: "runProjects", data, ...rest })),
                    ...(await shapeHelper({ relation: "steps", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    completedComplexity: noNull(data.completedComplexity),
                    contextSwitches: noNull(data.contextSwitches),
                    isPrivate: noNull(data.isPrivate),
                    status: noNull(data.status),
                    timeElapsed: noNull(data.timeElapsed),
                    ...(data.status === RunStatus.InProgress ? { startedAt: new Date() } : {}),
                    ...(data.status === RunStatus.Completed ? { completedAt: new Date() } : {}),
                    ...(await shapeHelper({ relation: "schedule", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "Schedule", parentRelationshipName: "runProjects", data, ...rest })),
                    ...(await shapeHelper({ relation: "steps", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "RunRoutineStep", parentRelationshipName: "runRoutine", data, ...rest })),
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
                { projectVersion: ProjectVersionModel.search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: {
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
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.run_projectSelect>(data, [
            ["organization", "Organization"],
            ["user", "User"],
        ], languages),
        profanityFields: ["name"],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
