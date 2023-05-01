import { Api, ApiCreateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput, apiValidation, ApiYou, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, labelShapeHelper, onCommonRoot, ownerShapeHelper, preShapeRoot, tagShapeHelper } from "../utils";
import { rootObjectDisplay } from "../utils/rootObjectDisplay";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { BookmarkModel } from "./bookmark";
import { OrganizationModel } from "./organization";
import { ReactionModel } from "./reaction";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";

const __typename = "Api" as const;
type Permissions = Pick<ApiYou, "canDelete" | "canUpdate" | "canBookmark" | "canTransfer" | "canRead" | "canReact">;
const suppFields = ["you"] as const;
export const ApiModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ApiCreateInput,
    GqlUpdate: ApiUpdateInput,
    GqlModel: Api,
    GqlPermission: Permissions,
    GqlSearch: ApiSearchInput,
    GqlSort: ApiSortBy,
    PrismaCreate: Prisma.apiUpsertArgs["create"],
    PrismaUpdate: Prisma.apiUpsertArgs["update"],
    PrismaModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    PrismaSelect: Prisma.apiSelect,
    PrismaWhere: Prisma.apiWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api,
    display: rootObjectDisplay(ApiVersionModel),
    format: {
        gqlRelMap: {
            __typename,
            createdBy: "User",
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
            },
            parent: "Api",
            tags: "Tag",
            versions: "ApiVersion",
            labels: "Label",
            issues: "Issue",
            pullRequests: "PullRequest",
            questions: "Question",
            bookmarkedBy: "User",
            stats: "StatsApi",
            transfers: "Transfer",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            ownedByUser: "User",
            ownedByOrganization: "Organization",
            parent: "ApiVersion",
            tags: "Tag",
            issues: "Issue",
            bookmarkedBy: "User",
            reactions: "Reaction",
            viewedBy: "View",
            pullRequests: "PullRequest",
            versions: "ApiVersion",
            labels: "Label",
            stats: "StatsApi",
            questions: "Question",
            transfers: "Transfer",
        },
        joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
        countFields: {
            issuesCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: true, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "forks", data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", relation: "labels", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: false, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", relation: "labels", data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonRoot({ ...params, objectType: __typename });
            },
        },
        yup: apiValidation,
    },
    search: {
        defaultSort: ApiSortBy.ScoreDesc,
        sortBy: ApiSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            issuesId: true,
            labelsIds: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
            pullRequestsId: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                "labelsWrapped",
                { versions: { some: "transNameWrapped" } },
                { versions: { some: "transSummaryWrapped" } },
            ],
        }),
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted === false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            versions: ["ApiVersion", ["root"]],
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
