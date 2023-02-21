import { MaxObjects, NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";
import { nodeRoutineListValidation } from '@shared/validation';
import { defaultPermissions } from '../utils';

const __typename = 'NodeRoutineList' as const;

const suppFields = [] as const;
export const NodeRoutineListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeRoutineListCreateInput,
    GqlUpdate: NodeRoutineListUpdateInput,
    GqlModel: NodeRoutineList,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_routine_listUpsertArgs['create'],
    PrismaUpdate: Prisma.node_routine_listUpsertArgs['update'],
    PrismaModel: Prisma.node_routine_listGetPayload<SelectWrap<Prisma.node_routine_listSelect>>,
    PrismaSelect: Prisma.node_routine_listSelect,
    PrismaWhere: Prisma.node_routine_listWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_routine_list,
    display: {
        select: () => ({ id: true, node: selPad(NodeModel.display.select) }),
        label: (select, languages) => NodeModel.display.label(select.node as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            items: 'NodeRoutineListItem',
        },
        prismaRelMap: {
            __typename,
            node: 'Node',
            items: 'NodeRoutineListItem',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                isOrdered: noNull(data.isOrdered),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: 'node', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'items', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeRoutineListItem', parentRelationshipName: 'list', data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                isOrdered: noNull(data.isOrdered),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: 'items', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeRoutineListItem', parentRelationshipName: 'list', data, prisma, userData })),
            }),
        },
        yup: nodeRoutineListValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ node: 'Node' }),
        permissionResolvers: defaultPermissions,
        owner: (data) => NodeModel.validate!.owner(data.node as any),
        isDeleted: (data, languages) => NodeModel.validate!.isDeleted(data.node as any, languages),
        isPublic: (data, languages) => NodeModel.validate!.isPublic(data.node as any, languages),
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
        }
    },
})