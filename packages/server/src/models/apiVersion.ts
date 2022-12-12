import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion, ApiVersionSearchInput, ApiVersionSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { Displayer, Formatter, Searcher } from "./types";

type Model = {
    GqlModel: ApiVersion,
    GqlSearch: ApiVersionSearchInput,
    GqlSort: ApiVersionSortBy,
    PrismaModel: Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>,
    PrismaSelect: Prisma.api_versionSelect,
    PrismaWhere: Prisma.api_versionWhereInput,
}

const __typename = 'ApiVersion' as const;

const suppFields = ['permissionsVersion'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        comments: 'Comment',
        directoryListings: 'ProjectVersionDirectory',
        forks: 'ApiVersion',
        reports: 'Report',
        resourceList: 'ResourceList',
        root: 'Api',
    },
    countFields: ['commentsCount', 'directoryListingsCount', 'forksCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['permissionsVersion', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: ApiVersionSortBy.DateUpdatedDesc,
    sortBy: ApiVersionSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'minScore',
        'minStars',
        'minViews',
        'ownedByOrganizationId',
        'ownedByUserId',
        'tags',
        'translationLanguages',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({
        OR: [
            'transSummaryWrapped',
            'transNameWrapped',
            { root: 'tagsWrapped' },
            { root: 'labelsWrapped' },
        ]
    }),
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => {
        // Return name if exists, or callLink host
        const name = bestLabel(select.translations, 'name', languages)
        if (name.length > 0) return name
        const url = new URL(select.callLink)
        return url.host
    },
})

export const ApiVersionModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api_version,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: searcher(),
    validate: {} as any,
})