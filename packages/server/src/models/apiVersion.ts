import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionSortBy, ApiVersionUpdateInput, PrependString, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { ModelLogic } from "./types";
import { ApiModel } from "./api";

const __typename = 'ApiVersion' as const;
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
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
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
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            ApiModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: 1000000,
        owner: (data) => ApiModel.validate!.owner(data.root as any),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: 'Api',
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            canComment: () => !isDeleted && (isAdmin || isPublic),
            canCopy: () => !isDeleted && (isAdmin || isPublic),
            canDelete: () => isAdmin && !isDeleted,
            canEdit: () => isAdmin && !isDeleted,
            canReport: () => !isAdmin && !isDeleted && isPublic,
            canRun: () => !isDeleted && (isAdmin || isPublic),
            canStar: () => !isDeleted && (isAdmin || isPublic),
            canUse: () => !isDeleted && (isAdmin || isPublic),
            canView: () => !isDeleted && (isAdmin || isPublic),
            canVote: () => !isDeleted && (isAdmin || isPublic),
        }),
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'Api',
                    prisma,
                    updateMany: updateMany as any,
                });
            },
            async create({ createMany, languages }) {
                createMany.forEach(input => lineBreaksCheck(input, ['summary'], 'LineBreaksBio', languages))
            },
            async update({ languages, updateMany }) {
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['summary'], 'LineBreaksBio', languages));
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
                root: ApiModel.validate!.visibility.owner(userId),
            }),
        },
    },
})