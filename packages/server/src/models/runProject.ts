import { MaxObjects, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectSortBy, RunProjectUpdateInput, runProjectValidation, RunProjectYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
import { ProjectVersionModel } from "./projectVersion";
import { ModelLogic } from "./types";

const __typename = "RunProject" as const;
type Permissions = Pick<RunProjectYou, "canDelete" | "canUpdate" | "canRead">;
const suppFields = ["you"] as const;
export const RunProjectModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunProjectCreateInput,
    GqlUpdate: RunProjectUpdateInput,
    GqlModel: RunProject,
    GqlSearch: RunProjectSearchInput,
    GqlSort: RunProjectSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.run_projectUpsertArgs["create"],
    PrismaUpdate: Prisma.run_projectUpsertArgs["update"],
    PrismaModel: Prisma.run_projectGetPayload<SelectWrap<Prisma.run_projectSelect>>,
    PrismaSelect: Prisma.run_projectSelect,
    PrismaWhere: Prisma.run_projectWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project,
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
    format: {
        gqlRelMap: {
            __typename,
            projectVersion: "ProjectVersion",
            schedule: "Schedule",
            steps: "RunProjectStep",
            user: "User",
            organization: "Organization",
        },
        prismaRelMap: {
            __typename,
            projectVersion: "ProjectVersion",
            schedule: "Schedule",
            steps: "RunProjectStep",
            user: "User",
            organization: "Organization",
        },
        countFields: {
            stepsCount: true,
        },
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
    mutate: {
        shape: {
            pre: async ({ updateList }) => {
                if (updateList.length) {
                    // TODO if status passed in for update, make sure it's not the same 
                    // as the current status, or an invalid transition (e.g. failed -> in progress)
                }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                //TODO
            } as any),
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
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { projectVersion: ProjectVersionModel.search!.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            organization: "Organization",
            projectVersion: "Routine",
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
