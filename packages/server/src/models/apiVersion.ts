import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionSortBy, ApiVersionUpdateInput, MaxObjects, PrependString, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, postShapeVersion, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { ModelLogic } from "./types";
import { ApiModel } from "./api";
import { apiVersionValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'ApiVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canUpdate' | 'canReport' | 'canUse' | 'canRead'>;
const suppFields = ['you'] as const;
export const ApiVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ApiVersionCreateInput,
    GqlUpdate: ApiVersionUpdateInput,
    GqlPermission: Permissions,
    GqlModel: ApiVersion,
    GqlSearch: ApiVersionSearchInput,
    GqlSort: ApiVersionSortBy,
    PrismaCreate: Prisma.api_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.api_versionUpsertArgs['update'],
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
            pullRequest: 'PullRequest',
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
            create: async ({ data, ...rest }) => ({
                id: data.id,
                callLink: data.callLink,
                documentationLink: noNull(data.documentationLink),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childApiVersions', data, ...rest })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'apiVersion', data, ...rest })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'Api', parentRelationshipName: 'versions', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                callLink: noNull(data.callLink),
                documentationLink: noNull(data.documentationLink),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childApiVersions', data, ...rest })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'apiVersion', data, ...rest })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Update'], isOneToOne: true, isRequired: false, objectType: 'Api', parentRelationshipName: 'versions', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            }
        },
        yup: apiVersionValidation,
    },
    search: {
        defaultSort: ApiVersionSortBy.DateUpdatedDesc,
        sortBy: ApiVersionSortBy,
        searchFields: {
            createdByIdRoot: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
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
        maxObjects: MaxObjects[__typename],
        owner: (data) => ApiModel.validate!.owner(data.root as any),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ['Api', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
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
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ]
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
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