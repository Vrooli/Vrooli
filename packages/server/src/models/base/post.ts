import { MaxObjects, PostSortBy, postValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { PreShapeEmbeddableTranslatableResult, preShapeEmbeddableTranslatable, tagShapeHelper, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { PostFormat } from "../formats";
import { PostModelLogic, TeamModelLogic } from "./types";

type PostPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Post" as const;
export const PostModel: PostModelLogic = ({
    __typename,
    dbTable: "post",
    dbTranslationTable: "post_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans?.description,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    }),
    format: PostFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<PostPre> => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as PostPre;
                return {
                    id: data.id,
                    isPinned: noNull(data.isPinned),
                    isPrivate: data.isPrivate,
                    team: data.teamConnect ? { connect: { id: data.teamConnect } } : undefined,
                    user: !data.teamConnect ? { connect: { id: rest.userData.id } } : undefined,
                    repostedFrom: await shapeHelper({ relation: "repostedFrom", relTypes: ["Connect"], isOneToOne: true, objectType: "Post", parentRelationshipName: "reposts", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Post", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as PostPre;
                return {
                    isPinned: noNull(data.isPinned),
                    isPrivate: noNull(data.isPrivate),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "post", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Post", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsPlain({
                    ...params,
                    objectType: __typename,
                    ownerTeamField: "team",
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
            teamId: true,
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
    validate: () => ({
        isDeleted: (data) => data.isDeleted === true,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data?.team,
            User: data?.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            team: "Team",
            user: "User",
        }),
        visibility: {
            private: function getVisibilityPrivate() {
                return {
                    isDeleted: false,
                    isPrivate: true,
                };
            },
            public: function getVisibilityPublic() {
                return {
                    isDeleted: false,
                    isPrivate: false,
                };
            },
            owner: (userId) => ({
                OR: [
                    { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { user: { id: userId } },
                ],
            }),
        },
    }),
});
