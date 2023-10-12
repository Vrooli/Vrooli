import { MaxObjects, NoteVersionSortBy, noteVersionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { NoteVersionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { NoteModelInfo, NoteModelLogic, NoteVersionModelInfo, NoteVersionModelLogic } from "./types";

const __typename = "NoteVersion" as const;
export const NoteVersionModel: NoteVersionModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.note_version,
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    name: trans?.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans?.description?.slice(0, 256),
                }, languages[0]);
            },
        },
    }),
    format: NoteVersionFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const { Create, Update, Delete, prisma, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    prisma,
                    Update,
                    userData,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const { translations } = await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest });
                const translationCreatesPromises = translations?.create?.map(async (translation) => ({
                    ...translation,
                    pagesCreate: undefined,
                    ...(await shapeHelper({ relation: "pages", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation, ...rest })),
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
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childNoteVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Note", parentRelationshipName: "versions", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => {
                // Translated pages require custom logic
                const { translations } = await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest });
                const translationCreatesPromises = translations?.create?.map(async (translation) => ({
                    ...translation,
                    pagesCreate: undefined,
                    ...(await shapeHelper({ relation: "pages", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation, ...rest })),
                }));
                const translationUpdatesPromises = translations?.update?.map(async (translation) => ({
                    where: translation.where,
                    data: {
                        ...translation.data,
                        pagesCreate: undefined,
                        pagesUpdate: undefined,
                        pagesDelete: undefined,
                        ...(await shapeHelper({ relation: "pages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NoteVersionPage" as any, parentRelationshipName: "translations", data: translation.data, ...rest })),
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
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childApiVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "Note", parentRelationshipName: "versions", data, ...rest })),
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
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByOrganizationIdRoot: true,
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
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
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
                root: ModelMap.get<NoteModelLogic>("Note").validate().visibility.owner(userId),
            }),
        },
    }),
});
