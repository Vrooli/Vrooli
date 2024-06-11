import { ApiVersionSortBy, apiVersionValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { afterMutationsVersion, preShapeVersion, PreShapeVersionResult, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { ApiVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ApiModelInfo, ApiModelLogic, ApiVersionModelInfo, ApiVersionModelLogic } from "./types";

type ApiVersionPre = PreShapeVersionResult;

const __typename = "ApiVersion" as const;
export const ApiVersionModel: ApiVersionModelLogic = ({
    __typename,
    dbTable: "api_version",
    dbTranslationTable: "api_version_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
            get: ({ callLink, translations }, languages) => {
                // Return name if exists, or callLink host
                const name = bestTranslation(translations, languages)?.name ?? "";
                if (name.length > 0) return name;
                const url = new URL(callLink);
                return url.host;
            },
        },
        embed: {
            select: () => ({
                id: true,
                callLink: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, summary: true } },
            }),
            get: ({ callLink, root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    callLink,
                    name: trans?.name,
                    summary: trans?.summary,
                    tags: (root as any).tags.map(({ tag }) => tag),
                }, languages[0]);
            },
        },
    }),
    format: ApiVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<ApiVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                    userData,
                });
                [...Create, ...Update].forEach(input => lineBreaksCheck(input, ["summary"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ApiVersionPre;
                return {
                    id: data.id,
                    callLink: data.callLink,
                    documentationLink: noNull(data.documentationLink),
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    schemaText: noNull(data.schemaText),
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childApiVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "apiVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Api", parentRelationshipName: "versions", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ApiVersionPre;
                return {
                    callLink: noNull(data.callLink),
                    documentationLink: noNull(data.documentationLink),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    schemaText: noNull(data.schemaText),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childApiVersions", data, ...rest }),
                    resourceList: await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "ResourceList", parentRelationshipName: "apiVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Api", parentRelationshipName: "versions", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: apiVersionValidation,
    },
    search: {
        defaultSort: ApiVersionSortBy.DateUpdatedDesc,
        sortBy: ApiVersionSortBy,
        searchFields: {
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
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transSummaryWrapped",
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
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<ApiVersionModelInfo["PrismaSelect"]>([["root", "Api"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ApiModelLogic>("Api").validate().owner(data?.root as ApiModelInfo["PrismaModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Api", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: {
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ],
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                root: ModelMap.get<ApiModelLogic>("Api").validate().visibility.owner(userId),
            }),
        },
    }),
});
