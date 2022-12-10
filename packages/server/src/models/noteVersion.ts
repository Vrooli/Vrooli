import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NoteVersion } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { Displayer, Formatter } from "./types";

const __typename = 'NoteVersion' as const;

const suppFields = ['permissionsVersion'] as const;
const formatter = (): Formatter<NoteVersion, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        comments: 'Comment',
        directoryListings: 'ProjectVersionDirectory',
        forks: 'NoteVersion',
        reports: 'Report',
        root: 'Note',
    },
    countFields: ['commentsCount', 'directoryListingsCount', 'forksCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['permissionsVersion', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const displayer = (): Displayer<
    Prisma.note_versionSelect,
    Prisma.note_versionGetPayload<SelectWrap<Prisma.note_versionSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const NoteVersionModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})