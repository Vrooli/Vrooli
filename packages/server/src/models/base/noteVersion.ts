import { MaxObjects, NoteVersionSortBy, noteVersionValidation, VersionYou } from "@local/shared";
import { noNull, shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, postShapeVersion, translationShapeHelper } from "../utils";
import { preShapeVersion } from "../utils/preShapeVersion";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { NoteModel } from "./note";
import { ModelLogic, NoteVersionModelLogic } from "./types";

const __typename = "NoteVersion" as const;
type Permissions = Pick<VersionYou, "canCopy" | "canDelete" | "canUpdate" | "canReport" | "canUse" | "canRead">;
const suppFields = ["you"] as const;
export const NoteVersionModel: ModelLogic<NoteVersionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, text: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    text: trans.text?.slice(0, 512),
                }, languages[0]);
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: "Comment",
            directoryListings: "ProjectVersionDirectory",
            pullRequest: "PullRequest",
            forks: "NoteVersion",
            reports: "Report",
            root: "Note",
        },
        prismaRelMap: {
            __typename,
            root: "Note",
            forks: "Note",
            pullRequest: "PullRequest",
            comments: "Comment",
            reports: "Report",
            directoryListings: "ProjectVersionDirectory",
        },
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async (params) => {
                const { createList, updateList, deleteList, prisma, userData } = params;
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: noNull(data.isPrivate),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childNoteVersions", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Note", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childApiVersions", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "Note", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
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
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            NoteModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => NoteModel.validate!.owner(data.root as any, userId),
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
                root: NoteModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
