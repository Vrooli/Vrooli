import { CodeVersionSortBy, MaxObjects, codeVersionValidation, getTranslation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { withRedis } from "../../redisConn";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { PreShapeVersionResult, afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { CodeVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { CodeModelInfo, CodeModelLogic, CodeVersionModelInfo, CodeVersionModelLogic } from "./types";

type CodeVersionPre = PreShapeVersionResult;

const __typename = "CodeVersion" as const;
export const CodeVersionModel: CodeVersionModelLogic = ({
    __typename,
    dbTable: "code_version",
    dbTranslationTable: "code_version_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans.description,
                }, languages[0]);
            },
        },
    }),
    format: CodeVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<CodeVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                    userData,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as CodeVersionPre;
                return {
                    id: data.id,
                    codeLanguage: data.codeLanguage,
                    codeType: data.codeType,
                    content: data.content,
                    default: noNull(data.default),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childCodeVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "codeVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Code", parentRelationshipName: "versions", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as CodeVersionPre;
                return {
                    content: noNull(data.content),
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childCodeVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "codeVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Code", parentRelationshipName: "versions", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ deletedIds, updatedIds, ...rest }) => {
                // We store the code contents in Redis for sandbox caching. Remove them
                const codeVersionIds = [...deletedIds, ...updatedIds];
                if (codeVersionIds.length > 0) {
                    await withRedis({
                        process: async (redisClient) => {
                            if (!redisClient) return;
                            await redisClient.del(codeVersionIds.map(id => `codeVersion:${id}`));
                        },
                        trace: "0646",
                    });
                }

                await afterMutationsVersion({ ...rest, deletedIds, updatedIds, objectType: __typename });
            },
        },
        yup: codeVersionValidation,
    },
    search: {
        defaultSort: CodeVersionSortBy.DateUpdatedDesc,
        sortBy: CodeVersionSortBy,
        searchFields: {
            calledByRoutineVersionId: true,
            codeLanguage: true,
            codeType: true,
            completedTimeFrame: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
            isLatest: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByTeamIdRoot: true,
            ownedByUserIdRoot: true,
            reportId: true,
            rootId: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<CodeVersionModelInfo["GqlPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<CodeVersionModelInfo["PrismaSelect"]>([["root", "Code"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<CodeModelLogic>("Code").validate().owner(data?.root as CodeModelInfo["PrismaModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Code", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: { isDeleted: false }, // Parent can't be deleted
                    OR: [ // Either the version, root, or the owner is private
                        { isPrivate: true },
                        { root: { isPrivate: true } },
                        { root: { ownedByTeam: useVisibility("Team", "private", ...params) } },
                        { root: { ownedByUser: { isPrivate: true } } },
                        { root: { ownedByUser: { isPrivateCodes: true } } },
                    ],
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Can't be private
                    root: {
                        isDeleted: false, // Root can't be deleted
                        isPrivate: false, // Root can't be private
                        OR: [ // Either the owner is public, or there is no owner
                            { ownedByTeam: null, ownedByUser: null },
                            { ownedByTeam: useVisibility("Team", "public", ...params) },
                            { ownedByUser: { isPrivate: false, isPrivateCodes: false } },
                        ],
                    },
                };
            },
            owner: (...params) => ({
                isDeleted: false, // Can't be deleted
                root: useVisibility("Code", "owner", ...params),
            }),
        },
    }),
});
