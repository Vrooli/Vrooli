import { Prisma } from "@prisma/client";
import { MaxObjects, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListSortBy, ResourceListUpdateInput } from "@shared/consts";
import { resourceListValidation } from "@shared/validation";
import { findFirstRel, shapeHelper, uppercaseFirstLetter } from "../builders";
import { SelectWrap } from "../builders/types";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, oneIsPublic, translationShapeHelper } from "../utils";
import { ApiModel } from "./api";
import { FocusModeModel } from "./focusMode";
import { OrganizationModel } from "./organization";
import { PostModel } from "./post";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { SmartContractModel } from "./smartContract";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";

const __typename = 'ResourceList' as const;
const suppFields = [] as const;
export const ResourceListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ResourceListCreateInput,
    GqlUpdate: ResourceListUpdateInput,
    GqlModel: ResourceList,
    GqlSearch: ResourceListSearchInput,
    GqlSort: ResourceListSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.resource_listUpsertArgs['create'],
    PrismaUpdate: Prisma.resource_listUpsertArgs['update'],
    PrismaModel: Prisma.resource_listGetPayload<SelectWrap<Prisma.resource_listSelect>>,
    PrismaSelect: Prisma.resource_listSelect,
    PrismaWhere: Prisma.resource_listWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.resource_list,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            resources: 'Resource',
            apiVersion: 'ApiVersion',
            organization: 'Organization',
            post: 'Post',
            projectVersion: 'ProjectVersion',
            routineVersion: 'RoutineVersion',
            smartContractVersion: 'SmartContractVersion',
            standardVersion: 'StandardVersion',
            focusMode: 'FocusMode',
        },
        prismaRelMap: {
            __typename,
            resources: 'Resource',
            apiVersion: 'ApiVersion',
            organization: 'Organization',
            post: 'Post',
            projectVersion: 'ProjectVersion',
            routineVersion: 'RoutineVersion',
            smartContractVersion: 'SmartContractVersion',
            standardVersion: 'StandardVersion',
            focusMode: 'FocusMode',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ...(await shapeHelper({ relation: 'apiVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'organization', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'post', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Post', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'projectVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ProjectVersion', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'smartContractVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'SmartContractVersion', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'standardVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'StandardVersion', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'focusMode', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'FocusMode', parentRelationshipName: 'resourceList', data, ...rest })),
                ...(await shapeHelper({ relation: 'resources', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'Resource', parentRelationshipName: 'list', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await shapeHelper({ relation: 'resources', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'Resource', parentRelationshipName: 'list', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            }),
        },
        yup: resourceListValidation,
    },
    search: {
        defaultSort: ResourceListSortBy.IndexAsc,
        sortBy: ResourceListSortBy,
        searchFields: {
            apiVersionId: true,
            createdTimeFrame: true,
            organizationId: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            smartContractVersionId: true,
            standardVersionId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            focusModeId: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'transNameWrapped',
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            apiVersion: 'ApiVersion',
            organization: 'Organization',
            post: 'Post',
            projectVersion: 'ProjectVersion',
            routineVersion: 'RoutineVersion',
            smartContractVersion: 'SmartContractVersion',
            standardVersion: 'StandardVersion',
            focusMode: 'FocusMode',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => {
            const [resourceOnType, resourceOnData] = findFirstRel(data, [
                'apiVersion',
                'focusMode',
                'organization',
                'post',
                'projectVersion',
                'routineVersion',
                'smartContractVersion',
                'standardVersion',
            ])
            const { validate } = getLogic(['validate'], uppercaseFirstLetter(resourceOnType!) as any, ['en'], 'ResourceListModel.validate.owner');
            return validate.owner(resourceOnData);
        },
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.resource_listSelect>(data, [
            ['apiVersion', 'Api'],
            ['focusMode', 'FocusMode'],
            ['organization', 'Organization'],
            ['post', 'Post'],
            ['projectVersion', 'Project'],
            ['routineVersion', 'Routine'],
            ['smartContractVersion', 'SmartContract'],
            ['standardVersion', 'Standard'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { apiVersion: ApiModel.validate!.visibility.owner(userId) },
                    { focusMode: FocusModeModel.validate!.visibility.owner(userId) },
                    { organization: OrganizationModel.validate!.visibility.owner(userId) },
                    { post: PostModel.validate!.visibility.owner(userId) },
                    { project: ProjectModel.validate!.visibility.owner(userId) },
                    { routineVersion: RoutineModel.validate!.visibility.owner(userId) },
                    { smartContractVersion: SmartContractModel.validate!.visibility.owner(userId) },
                    { standardVersion: StandardModel.validate!.visibility.owner(userId) },
                ]
            }),
        }
    },
})