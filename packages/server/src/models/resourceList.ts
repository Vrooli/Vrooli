import { resourceListValidation } from "@shared/validation";
import { MaxObjects, ResourceListSortBy } from "@shared/consts";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput, SessionUser } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { bestLabel, defaultPermissions, oneIsPublic, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { ApiModel } from "./api";
import { PostModel } from "./post";
import { UserScheduleModel } from "./userSchedule";
import { SmartContractModel } from "./smartContract";
import { shapeHelper } from "../builders";

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        // organization: data.organizationId ? { connect: { id: data.organizationId } } : undefined,
        // project: data.projectId ? { connect: { id: data.projectId } } : undefined,
        // routine: data.routineId ? { connect: { id: data.routineId } } : undefined,
        // user: data.userId ? { connect: { id: data.userId } } : undefined,
        // resources: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'resources', objectType: 'Resource', prisma, userData }),
        // translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

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
            userSchedule: 'UserSchedule',
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
            userSchedule: 'UserSchedule',
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
                ...(await shapeHelper({ relation: 'userSchedule', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'UserSchedule', parentRelationshipName: 'resourceList', data, ...rest })),
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
            userScheduleId: true,
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
            userSchedule: 'UserSchedule',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.organization,
            User: (data.userSchedule as any)?.user,
        }), // TODO this is incorrect. Should be owner of apiVersion, organization, post, etc.
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.resource_listSelect>(data, [
            ['apiVersion', 'Api'],
            ['organization', 'Organization'],
            ['post', 'Post'],
            ['projectVersion', 'Project'],
            ['routineVersion', 'Routine'],
            ['smartContractVersion', 'SmartContract'],
            ['standardVersion', 'Standard'],
            ['userSchedule', 'UserSchedule'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { apiVersion: ApiModel.validate!.visibility.owner(userId) },
                    { organization: OrganizationModel.validate!.visibility.owner(userId) },
                    { post: PostModel.validate!.visibility.owner(userId) },
                    { project: ProjectModel.validate!.visibility.owner(userId) },
                    { routineVersion: RoutineModel.validate!.visibility.owner(userId) },
                    { smartContractVersion: SmartContractModel.validate!.visibility.owner(userId) },
                    { standardVersion: StandardModel.validate!.visibility.owner(userId) },
                    { userSchedule: UserScheduleModel.validate!.visibility.owner(userId) },
                ]
            }),
        }
    },
})