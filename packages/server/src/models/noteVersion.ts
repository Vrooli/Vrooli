import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionSortBy, NoteVersionUpdateInput, VersionPermission } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ModelLogic } from "./types";

const __typename = 'NoteVersion' as const;
const suppFields = ['permissionsVersion'] as const;
export const NoteVersionModel: ModelLogic<{
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
            toGraphQL: ({ ids, prisma, userData }) => ({
                permissionsVersion: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
            }),
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})