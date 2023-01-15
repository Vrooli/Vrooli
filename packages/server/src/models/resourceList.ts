import { resourceListValidation } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput, SessionUser } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { permissionsSelectHelper } from "../builders";
import { bestLabel, oneIsPublic } from "../utils";
import { SelectWrap } from "../builders/types";
import { ApiModel } from "./api";
import { PostModel } from "./post";
import { UserScheduleModel } from "./userSchedule";
import { SmartContractModel } from "./smartContract";

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
    GqlPermission: any,
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
            create: async ({ data, prisma, userData }) => {
                return {
                    ...await shapeBase(prisma, userData, data, true),
                };
            },
            update: async ({ data, prisma, userData }) => {
                return {
                    ...await shapeBase(prisma, userData, data, false),
                };
            },
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
        maxObjects: 50000,
        permissionsSelect: (...params) => ({
            id: true,
            ...permissionsSelectHelper([
                // ['apiVersion', 'Api'],
                ['organization', 'Organization'],
                // ['post', 'Post'],
                ['projectVersion', 'Project'],
                ['routineVersion', 'Routine'],
                // ['smartContractVersion', 'SmartContract'],
                ['standardVersion', 'Standard'],
                // ['userSchedule', 'UserSchedule'],
            ], ...params)
        }),
        permissionResolvers: ({ isAdmin }) => ({
            canDelete: () => isAdmin,
            canEdit: () => isAdmin,
        }),
        owner: (data) => ({
            Organization: data.organization,
            User: (data.userSchedule as any)?.user,
        }),
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