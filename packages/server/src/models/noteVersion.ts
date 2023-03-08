import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionSortBy, NoteVersionUpdateInput, PrependString, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { ModelLogic } from "./types";
import { NoteModel } from "./note";
import { noteVersionValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'NoteVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canUpdate' | 'canReport' | 'canUse' | 'canRead'>;
const suppFields = ['you'] as const;
export const NoteVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NoteVersionCreateInput,
    GqlUpdate: NoteVersionUpdateInput,
    GqlModel: NoteVersion,
    GqlSearch: NoteVersionSearchInput,
    GqlSort: NoteVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.note_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.note_versionUpsertArgs['update'],
    PrismaModel: Prisma.note_versionGetPayload<SelectWrap<Prisma.note_versionSelect>>,
    PrismaSelect: Prisma.note_versionSelect,
    PrismaWhere: Prisma.note_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'NoteVersion',
            reports: 'Report',
            root: 'Note',
        },
        prismaRelMap: {
            __typename,
            root: 'Note',
            forks: 'Note',
            pullRequest: 'PullRequest',
            comments: 'Comment',
            reports: 'Report',
            directoryListings: 'ProjectVersionDirectory',
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
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ prisma, userData, data }) => ({
                id: data.id,
                isLatest: noNull(data.isLatest),
                isPrivate: noNull(data.isPrivate),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childNoteVersions', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'Note', parentRelationshipName: 'versions', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
            }),
            update: async ({ prisma, userData, data }) => ({
                isLatest: noNull(data.isLatest),
                isPrivate: noNull(data.isPrivate),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childApiVersions', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Update'], isOneToOne: true, isRequired: false, objectType: 'Note', parentRelationshipName: 'versions', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
            }),
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'transNameWrapped',
                { root: 'tagsWrapped' },
                { root: 'labelsWrapped' },
            ]
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => data.isPrivate === false &&
            NoteModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => NoteModel.validate!.owner(data.root as any),
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            root: ['Note', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'Note',
                    prisma,
                    updateMany: updateMany as any,
                });
            },
            async create({ createMany, languages }) {
                createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksBio', languages))
            },
            async update({ languages, updateMany }) {
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['description'], 'LineBreaksBio', languages));
            },
        },
        visibility: {
            private: {
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ]
            },
            public: {
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ]
            },
            owner: (userId) => ({
                root: NoteModel.validate!.visibility.owner(userId),
            }),
        },
    },
})