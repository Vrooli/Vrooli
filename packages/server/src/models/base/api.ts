import { ApiSortBy, apiValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils/defaultPermissions";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers/afterMutationsRoot";
import { getSingleTypePermissions } from "../../validators/permissions";
import { ApiFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ApiModelLogic, ApiVersionModelLogic, BookmarkModelLogic, OrganizationModelLogic, ReactionModelLogic, ViewModelLogic } from "./types";

const __typename = "Api" as const;
export const ApiModel: ApiModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.api,
    display: () => rootObjectDisplay(ModelMap.get<ApiVersionModelLogic>("ApiVersion")),
    format: ApiFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: data.isPrivate,
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: true, objectType: __typename, data, ...rest })),
                parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "ApiVersion", parentRelationshipName: "forks", data, ...rest }),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Api", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "apis", isCreate: false, objectType: __typename, data, ...rest })),
                versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "root", data, ...rest }),
                tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", relation: "tags", data, ...rest }),
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Api", data, ...rest }),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
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
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted === false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data?.ownedByOrganization,
            User: data?.ownedByUser,
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
                    { ownedByOrganization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
