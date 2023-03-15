import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionSearchInput, SmartContractVersionSortBy, SmartContractVersionUpdateInput, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, postShapeVersion, translationShapeHelper } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { SmartContractModel } from "./smartContract";
import { smartContractVersionValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

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
    mutate: {
        shape: {
            pre: async ({ createList, updateList, deleteList, prisma, userData }) => {
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksBio', userData.languages));
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                content: data.content,
                contractType: data.contractType,
                default: noNull(data.default),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childSmartContractVersions', data, ...rest })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'smartContractVersion', data, ...rest })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'SmartContract', parentRelationshipName: 'versions', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                content: noNull(data.content),
                contractType: noNull(data.contractType),
                default: noNull(data.default),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: 'directoryListings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'childSmartContractVersions', data, ...rest })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'smartContractVersion', data, ...rest })),
                ...(await shapeHelper({ relation: 'root', relTypes: ['Update'], isOneToOne: true, isRequired: true, objectType: 'SmartContract', parentRelationshipName: 'versions', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            }
        },
        yup: smartContractVersionValidation,
    },
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
                root: SmartContractModel.validate!.visibility.owner(userId),
            }),
        },
    },
})