import { MaxObjects, PostSortBy, postValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, getEmbeddableString, onCommonPlain, tagShapeHelper } from "../../utils";
import { preShapeEmbeddableTranslatable } from "../../utils/preShapeEmbeddableTranslatable";
import { PostFormat } from "../format/post";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { PostModelLogic } from "./types";

const __typename = "Post" as const;
const suppFields = [] as const;
export const PostModel: ModelLogic<PostModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.post,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages[0]);
            },
        },
    },
    format: PostFormat,
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
