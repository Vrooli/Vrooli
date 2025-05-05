import { DEFAULT_LANGUAGE, generatePublicId, getTranslation, MaxObjects, ResourceSortBy, resourceValidation, ResourceVersion, Tag } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { getLabels } from "../../getters/getLabels.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { ownerFields } from "../../utils/shapes/ownerFields.js";
import { preShapeRoot, type PreShapeRootResult } from "../../utils/shapes/preShapeRoot.js";
import { tagShapeHelper } from "../../utils/shapes/tagShapeHelper.js";
import { afterMutationsRoot } from "../../utils/triggers/afterMutationsRoot.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { ResourceFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, ReactionModelLogic, ResourceModelInfo, ResourceModelLogic, ViewModelLogic } from "./types.js";

type ResourcePre = PreShapeRootResult;

const __typename = "Resource" as const;
export const ResourceModel: ResourceModelLogic = ({
    __typename,
    dbTable: "resource",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                isDeleted: false,
                OR: [
                    {
                        isPrivate: false,
                        versions: { select: { id: true, isLatestPublic: true, translations: { select: { language: true, name: true } } } },

                    },
                    {
                        isPrivate: true,
                        versions: { select: { id: true, isLatest: true, translations: { select: { language: true, name: true } } } },
                    },
                ],
            }),
            get: (select, languages) => {
                const latestVersion = select.versions.find((version) => version.isLatestPublic || version.isLatest);
                return getTranslation(latestVersion as unknown as ResourceVersion, languages).name ?? "";
            },
        },
        embed: {
            select: () => ({
                id: true,
                isDeleted: false,
                tags: { select: { tag: true } },
                OR: [
                    {
                        isPrivate: false,
                        versions: { select: { id: true, isLatestPublic: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } } },
                    },
                    {
                        isPrivate: true,
                        versions: { select: { id: true, isLatest: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } } },
                    },
                ],
            }),
            get: ({ tags, versions }, languages) => {
                const latestVersion = versions.find((version) => version.isLatestPublic || version.isLatest);
                const trans = getTranslation(latestVersion as unknown as ResourceVersion, languages);
                return getEmbeddableString({
                    name: trans.name,
                    tags: (tags as unknown as Tag[]).map(({ tag }) => tag),
                    description: trans.description,
                }, languages?.[0]);
            },
        },
    }),
    format: ResourceFormat,
    mutate: {
        shape: {
            // TODO for morning 2: need to create helper to handle version pre/post logic. These should 
            // also support calling Trigger, and also version index logic. I started implementing this (the version index logic) somewhere before, 
            // maybe models/resourceVersion.
            pre: async (params): Promise<ResourcePre> => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ResourcePre;
                return {
                    id: BigInt(data.id),
                    publicId: generatePublicId(),
                    isInternal: noNull(data.isInternal),
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "resources", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "ResourceVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "ResourceVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Resource", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ResourcePre;
                return {
                    isInternal: noNull(data.isInternal),
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "resources", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ResourceVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Resource", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
            },
        },
        yup: resourceValidation,
    },
    search: {
        defaultSort: ResourceSortBy.ScoreDesc,
        sortBy: ResourceSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            isInternal: true,
            issuesId: true,
            latestVersionResourceSubType: true,
            latestVersionResourceSubTypes: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByTeamId: true,
            ownedByUserId: true,
            parentId: true,
            pullRequestsId: true,
            resourceType: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                { versions: { some: "transDescriptionWrapped" } },
                { versions: { some: "transNameWrapped" } },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ResourceModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                    "translatedName": await getLabels(ids, __typename, userData?.languages ?? [DEFAULT_LANGUAGE], "resource.translatedName"),
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, ...rest) =>
            data.isPrivate === false &&
            data.isDeleted === false &&
            data.isInternal === false &&
            (
                (data.ownedByUser === null && data.ownedByTeam === null) ||
                oneIsPublic<ResourceModelInfo["DbSelect"]>([
                    ["ownedByTeam", "Team"],
                    ["ownedByUser", "User"],
                ], data, ...rest)
            ),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data?.ownedByTeam,
            User: data?.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isInternal: true,
            isPrivate: true,
            permissions: true,
            createdBy: "User",
            ownedByTeam: "Team",
            ownedByUser: "User",
            versions: ["ResourceVersion", ["root"]],
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isInternal: false, // Internal resources should never be in search results
                    OR: [
                        { ownedByTeam: useVisibility("Team", "Own", data) },
                        { ownedByUser: useVisibility("User", "Own", data) },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isInternal: false, // Internal resources should never be in search results
                    OR: [
                        { ownedByTeam: null, ownedByUser: null },
                        { ownedByTeam: useVisibility("Team", "Own", data) },
                        { ownedByUser: useVisibility("User", "Own", data) },
                        { ownedByTeam: useVisibility("Team", "Public", data) },
                        { ownedByUser: useVisibility("User", "Public", data) },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isInternal: false, // Internal resources should never be in search results
                    isPrivate: true,  // Must be private
                    OR: (useVisibility("Resource", "Own", data) as { OR: object[] }).OR,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Must be public
                    OR: (useVisibility("Resource", "Own", data) as { OR: object[] }).OR,
                };
            },
            public: function getPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isInternal: false, // Internal resources should never be in search results
                    isPrivate: false, // Can't be private
                    OR: [
                        { ownedByTeam: null, ownedByUser: null },
                        { ownedByTeam: useVisibility("Team", "Public", data) },
                        { ownedByUser: { isPrivate: false, isPrivateResources: false } },
                    ],
                };
            },
        },
    }),
});
