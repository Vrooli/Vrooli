import { MaxObjects, Post, PostCreateInput, PostSearchInput, PostSortBy, PostUpdateInput, postValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, onCommonPlain, tagShapeHelper } from "../../utils";
import { preShapeEmbeddableTranslatable } from "../../utils/preShapeEmbeddableTranslatable";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Post" as const;
export const PostFormat: Formatter<ModelPostLogic> = {
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
            pre: async ({ createList, updateList }) => {
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                organization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
                user: !data.organizationConnect ? { connect: { id: rest.userData.id } } : undefined,
                ...(await shapeHelper({ relation: "repostedFrom", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Post", parentRelationshipName: "reposts", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Post", relation: "tags", data, ...rest })),
                // ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Post", relation: "tags", data, ...rest })),
                // ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: "organization",
                    ownerUserField: "user",
                });
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
};
