import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionSortBy, NoteVersionUpdateInput, VersionPermission } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { Displayer, Formatter, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NoteVersionCreateInput,
    GqlUpdate: NoteVersionUpdateInput,
    GqlModel: NoteVersion,
    GqlSearch: NoteVersionSearchInput,
    GqlSort: NoteVersionSortBy,
    GqlPermission: VersionPermission,
    PrismaCreate: Prisma.note_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.note_versionUpsertArgs['update'],
    PrismaModel: Prisma.note_versionGetPayload<SelectWrap<Prisma.note_versionSelect>>,
    PrismaSelect: Prisma.note_versionSelect,
    PrismaWhere: Prisma.note_versionWhereInput,
}

const __typename = 'NoteVersion' as const;

const suppFields = ['permissionsVersion'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
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
    countFields: ['commentsCount', 'directoryListingsCount', 'forksCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => ({
            permissionsVersion: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
        }),
    },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const NoteVersionModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})