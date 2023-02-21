import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, PrependString, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionSearchInput, SmartContractVersionSortBy, SmartContractVersionUpdateInput, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { SmartContractModel } from "./smartContract";

const __typename = 'SmartContractVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canUpdate' | 'canReport' | 'canUse' | 'canRead'>;
const suppFields = ['you'] as const;
export const SmartContractVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: SmartContractVersionCreateInput,
    GqlUpdate: SmartContractVersionUpdateInput,
    GqlModel: SmartContractVersion,
    GqlSearch: SmartContractVersionSearchInput,
    GqlSort: SmartContractVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.smart_contract_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.smart_contract_versionUpsertArgs['update'],
    PrismaModel: Prisma.smart_contract_versionGetPayload<SelectWrap<Prisma.smart_contract_versionSelect>>,
    PrismaSelect: Prisma.smart_contract_versionSelect,
    PrismaWhere: Prisma.smart_contract_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract_version,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'SmartContractVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'SmartContract',
        },
        prismaRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'SmartContractVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'SmartContract',
        },
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
            translationsCount: true,
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
    mutate: {} as any,
    search: {
        defaultSort: SmartContractVersionSortBy.DateUpdatedDesc,
        sortBy: SmartContractVersionSortBy,
        searchFields: {
            completedTimeFrame: true,
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
            reportId: true,
            rootId: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            userId: true,
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
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            SmartContractModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => SmartContractModel.validate!.owner(data.root as any),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ['SmartContract', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'SmartContract',
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
                root: SmartContractModel.validate!.visibility.owner(userId),
            }),
        },
    },
})