import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionSortBy, ApiVersionUpdateInput, PrependString, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ModelLogic } from "./types";

const type = 'ApiVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canEdit' | 'canReport' | 'canUse' | 'canView'>;
const suppFields = ['you.canCopy', 'you.canDelete', 'you.canEdit', 'you.canReport', 'you.canUse', 'you.canView'] as const;
export const ApiVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ApiVersionCreateInput,
    GqlUpdate: ApiVersionUpdateInput,
    GqlPermission: Permissions,
    GqlModel: ApiVersion,
    GqlSearch: ApiVersionSearchInput,
    GqlSort: ApiVersionSortBy,
    PrismaCreate: Prisma.api_keyUpsertArgs['create'],
    PrismaUpdate: Prisma.api_keyUpsertArgs['update'],
    PrismaModel: Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>,
    PrismaSelect: Prisma.api_versionSelect,
    PrismaWhere: Prisma.api_versionWhereInput,
}, typeof suppFields> = ({
    type,
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
            type,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'ApiVersion',
            reports: 'Report',
            resourceList: 'ResourceList',
            root: 'Api',
        },
        prismaRelMap: {
            type,
            calledByRoutineVersions: 'RoutineVersion',
            comments: 'Comment',
            reports: 'Report',
            root: 'Api',
            forks: 'Api',
            resourceList: 'ResourceList',
            pullRequest: 'PullRequest',
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
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: ApiVersionSortBy.DateUpdatedDesc,
        sortBy: ApiVersionSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minStars: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
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