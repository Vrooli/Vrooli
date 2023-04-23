import { MaxObjects, RunProjectSortBy } from "@local/consts";
import { runProjectValidation } from "@local/validation";
import { defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
const __typename = "RunProject";
const suppFields = ["you"];
export const RunProjectModel = ({
    __typename,
    delegate: (prisma) => prisma.run_project,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
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
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ updateList }) => {
                if (updateList.length) {
                }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
            }),
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
            startedTimeFrame: true,
            status: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { projectVersion: RunProjectModel.search.searchStringQuery() },
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
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic(data, [
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
//# sourceMappingURL=runProject.js.map