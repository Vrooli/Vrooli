import { MaxObjects, PostSortBy } from "@local/consts";
import { postValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, onCommonPlain, tagShapeHelper } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "Post";
const suppFields = [];
export const PostModel = ({
    __typename,
    delegate: (prisma) => prisma.post,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: "Comment",
            owner: {
                user: "User",
                organization: "Organization",
            },
            reports: "Report",
            repostedFrom: "Post",
            reposts: "Post",
            resourceList: "ResourceList",
            bookmarkedBy: "User",
            tags: "Tag",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
            repostedFrom: "Post",
            reposts: "Post",
            resourceList: "ResourceList",
            comments: "Comment",
            bookmarkedBy: "User",
            reactions: "Reaction",
            viewedBy: "View",
            reports: "Report",
            tags: "Tag",
        },
        countFields: {
            commentsCount: true,
            repostsCount: true,
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                organization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
                user: !data.organizationConnect ? { connect: { id: rest.userData.id } } : undefined,
                ...(await shapeHelper({ relation: "repostedFrom", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Post", parentRelationshipName: "reposts", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Post", relation: "tags", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Post", relation: "tags", data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: "organization",
                    ownerUserField: "user",
                });
            },
        },
        yup: postValidation,
    },
    search: {
        defaultSort: PostSortBy.ScoreDesc,
        sortBy: PostSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            isPinned: true,
            maxScore: true,
            maxBookmarks: true,
            minScore: true,
            minBookmarks: true,
            organizationId: true,
            userId: true,
            repostedFromIds: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "nameWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted === true,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            organization: "Organization",
            user: "User",
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
//# sourceMappingURL=post.js.map