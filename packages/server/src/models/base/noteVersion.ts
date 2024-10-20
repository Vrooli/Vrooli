import { MaxObjects, NoteVersionSortBy, getTranslation, noteVersionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { PreShapeVersionResult, afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { NoteVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { NoteModelInfo, NoteModelLogic, NoteVersionModelInfo, NoteVersionModelLogic } from "./types";

type NoteVersionPre = PreShapeVersionResult;

const __typename = "NoteVersion" as const;
export const NoteVersionModel: NoteVersionModelLogic = ({
    __typename,
    dbTable: "note_version",
    dbTranslationTable: "note_version_translation",
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
                    description: trans.description?.slice(0, 256),
                }, languages[0]);
            },
        },
    }),
    format: NoteVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<NoteVersionPre> => {
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
                const preData = rest.preMap[__typename] as NoteVersionPre;
                const translations = await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest });
                const translationCreatesPromises = translations?.create?.map(async (translation) => ({
                    ...translation,
                    pagesCreate: undefined,
                    pages: await shapeHelper({ relation: "pages", relTypes: ["Create"], isOneToOne: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation, ...rest }),
                }));
                const translationCreates = await Promise.all(translationCreatesPromises ?? []);
                return {
                    id: data.id,
                    isPrivate: data.isPrivate,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    translations: {
                        create: translationCreates,
                    },
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childNoteVersions", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Note", parentRelationshipName: "versions", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                // Translated pages require custom logic
                const preData = rest.preMap[__typename] as NoteVersionPre;
                const translations = await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest });
                const translationCreatesPromises = translations?.create?.map(async (translation) => ({
                    ...translation,
                    pagesCreate: undefined,
                    pages: await shapeHelper({ relation: "pages", relTypes: ["Create"], isOneToOne: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation, ...rest }),
                }));
                const translationUpdatesPromises = translations?.update?.map(async (translation) => ({
                    where: translation.where,
                    data: {
                        ...translation.data,
                        pagesCreate: undefined,
                        pagesUpdate: undefined,
                        pagesDelete: undefined,
                        pages: await shapeHelper({ relation: "pages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation.data, ...rest }),
                    },
                }));
                const translationCreates = await Promise.all(translationCreatesPromises ?? []);
                const translationUpdates = await Promise.all(translationUpdatesPromises ?? []);
                const translationDeletes = translations?.delete;
                return {
                    isPrivate: noNull(data.isPrivate),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    translations: {
                        create: translationCreates,
                        update: translationUpdates,
                        delete: translationDeletes,
                    },
                    directoryListings: await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childApiVersions", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Note", parentRelationshipName: "versions", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: noteVersionValidation,
    },
    search: {
        defaultSort: NoteVersionSortBy.DateUpdatedDesc,
        sortBy: NoteVersionSortBy,
        searchFields: {
            createdByIdRoot: true,
            createdTimeFrame: true,
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
                        ...(await getSingleTypePermissions<NoteVersionModelInfo["GqlPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            oneIsPublic<NoteVersionModelInfo["PrismaSelect"]>([["root", "Note"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<NoteModelLogic>("Note").validate().owner(data?.root as NoteModelInfo["PrismaModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Note", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: useVisibility("Note", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Objects you own
                        {
                            root: useVisibility("Note", "Own", data),
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            root: (useVisibility("NoteVersion", "Public", data) as { root: object }).root,
                        },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Private versions you own
                        {
                            isPrivate: true, // Version is private
                            root: useVisibility("Note", "Own", data),
                        },
                        // Private roots you own
                        {
                            root: {
                                isPrivate: true, // Root is private
                                ...useVisibility("Api", "Own", data),
                            },
                        },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Public versions you own
                        {
                            isPrivate: false, // Version is public
                            root: useVisibility("Note", "Own", data),
                        },
                        // Public roots you own
                        {
                            root: {
                                isPrivate: false, // Root is public
                                ...useVisibility("Note", "Own", data),
                            },
                        },
                    ],
                };
            },
            public: function getPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Version can't be private
                    root: {
                        isDeleted: false, // Root can't be deleted
                        isPrivate: false, // Root can't be private
                        OR: [
                            // Unowned
                            { ownedByTeam: null, ownedByUser: null },
                            // Owned by public teams
                            { ownedByTeam: useVisibility("Team", "Public", data) },
                            // Owned by public users
                            { ownedByUser: { isPrivate: false, isPrivateNotes: false } },
                        ],
                    },
                };
            },
        },
    }),
});
