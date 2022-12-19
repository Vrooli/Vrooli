import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionSortBy, ApiVersionUpdateInput, VersionPermission } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ModelLogic } from "./types";

const __typename = 'ApiVersion' as const;
const suppFields = ['permissionsVersion'] as const;
export const ApiVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ApiVersionCreateInput,
    GqlUpdate: ApiVersionUpdateInput,
    GqlPermission: VersionPermission,
    GqlModel: ApiVersion,
    GqlSearch: ApiVersionSearchInput,
    GqlSort: ApiVersionSortBy,
    PrismaCreate: Prisma.api_keyUpsertArgs['create'],
    PrismaUpdate: Prisma.api_keyUpsertArgs['update'],
    PrismaModel: Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>,
    PrismaSelect: Prisma.api_versionSelect,
    PrismaWhere: Prisma.api_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api_version,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => {
            // Return name if exists, or callLink host
            const name = bestLabel(select.translations, 'name', languages)
            if (name.length > 0) return name
            const url = new URL(select.callLink)
            return url.host
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'ApiVersion',
            reports: 'Report',
            resourceList: 'ResourceList',
            root: 'Api',
        },
        prismaRelMap: {
            __typename,
            calledByRoutineVersions: 'RoutineVersion',
            comments: 'Comment',
            reports: 'Report',
            root: 'Api',
            forks: 'Api',
            resourceList: 'ResourceList',
            pullRequest: 'PullRequest',
            directoryListings: 'ProjectVersionDirectory',
        },
        countFields: ['commentsCount', 'directoryListingsCount', 'forksCount', 'reportsCount'],
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: ({ ids, prisma, userData }) => ({
                permissionsVersion: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
            }),
        },
    },
    mutate: {} as any,
    search: {
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
    },
    validate: {} as any,
})