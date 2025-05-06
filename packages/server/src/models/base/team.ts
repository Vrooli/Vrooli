import { DEFAULT_LANGUAGE, MaxObjects, TeamSortBy, generatePublicId, getTranslation, teamValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { getLabels } from "../../getters/getLabels.js";
import { EmbeddingService } from "../../services/embedding.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { tagShapeHelper } from "../../utils/shapes/tagShapeHelper.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { handlesCheck } from "../../validators/handlesCheck.js";
import { lineBreaksCheck } from "../../validators/lineBreaksCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { TeamFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, TeamModelInfo, TeamModelLogic, ViewModelLogic } from "./types.js";

type TeamPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Team" as const;
export const TeamModel: TeamModelLogic = ({
    __typename,
    dbTable: "team",
    dbTranslationTable: "team_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, bio: true } } }),
            get: ({ translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return EmbeddingService.getEmbeddableString({
                    bio: trans.bio,
                    name: trans.name,
                }, languages?.[0]);
            },
        },
    }),
    format: TeamFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<TeamPre> => {
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["bio"], "LineBreaksBio"));
                await handlesCheck(__typename, Create, Update);
                // Find translations that need text embeddings
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    id: BigInt(data.id),
                    publicId: generatePublicId(),
                    bannerImage: data.bannerImage,
                    config: noNull(data.config),
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: data.isPrivate,
                    profileImage: data.profileImage,
                    createdBy: { connect: { id: BigInt(rest.userData.id) } },
                    members: {
                        create: {
                            publicId: generatePublicId(),
                            isAdmin: true,
                            user: { connect: { id: BigInt(rest.userData.id) } },
                        },
                    },
                    memberInvites: await shapeHelper({ relation: "memberInvites", relTypes: ["Create"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TeamPre;
                return {
                    bannerImage: noNull(data.bannerImage),
                    config: noNull(data.config),
                    handle: noNull(data.handle),
                    isOpenToNewMembers: noNull(data.isOpenToNewMembers),
                    isPrivate: noNull(data.isPrivate),
                    profileImage: noNull(data.profileImage),
                    members: await shapeHelper({ relation: "members", relTypes: ["Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    memberInvites: await shapeHelper({ relation: "memberInvites", relTypes: ["Create", "Delete"], isOneToOne: false, objectType: "Member", parentRelationshipName: "team", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Team", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                for (const teamId of createdIds) {
                    // Handle trigger
                    // Trigger(userData.languages).createTeam(userData.id, teamId);
                }
            },
        },
        yup: teamValidation,
    },
    search: {
        defaultSort: TeamSortBy.BookmarksDesc,
        searchFields: {
            createdTimeFrame: true,
            isOpenToNewMembers: true,
            maxBookmarks: true,
            maxViews: true,
            memberUserIds: true,
            minBookmarks: true,
            minViews: true,
            reportId: true,
            resourceId: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: TeamSortBy,
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                "transNameWrapped",
                "transBioWrapped",
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<TeamModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, userData?.languages ?? [DEFAULT_LANGUAGE], "team.translatedName"),
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canAddMembers: () => isLoggedIn && isAdmin,
        }),
        permissionsSelect: (userId) => ({
            id: true,
            isOpenToNewMembers: true,
            isPrivate: true,
            languages: true,
            permissions: true,
            ...(userId ? {
                members: {
                    where: {
                        userId: BigInt(userId),
                    },
                    select: {
                        id: true,
                        isAdmin: true,
                        permissions: true,
                        userId: true,
                    },
                },
            } : {}),
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    members: {
                        some: {
                            isAdmin: true,
                            user: { id: BigInt(data.userId) },
                        },
                    },
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { members: { some: { isAdmin: true, user: { id: BigInt(data.userId) } } } },
                        { isPrivate: false },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    members: {
                        some: {
                            isAdmin: true,
                            user: { id: BigInt(data.userId) },
                        },
                    },
                    isPrivate: true,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    members: {
                        some: {
                            isAdmin: true,
                            user: { id: BigInt(data.userId) },
                        },
                    },
                    isPrivate: false,
                };
            },
            public: function getPublic() {
                return {
                    isPrivate: false,
                };
            },
        },
    }),
});
