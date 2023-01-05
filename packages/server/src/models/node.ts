import { Node, NodeCreateInput, NodeUpdateInput } from '@shared/consts';
import { nodeValidation } from "@shared/validation";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { CustomError } from "../events";
import { RoutineModel } from "./routine";
import { bestLabel, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { noNull, shapeHelper } from "../builders";

const __typename = 'Node' as const;
const MAX_NODES_IN_ROUTINE = 100;
const suppFields = [] as const;
export const NodeModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeCreateInput,
    GqlUpdate: NodeUpdateInput,
    GqlModel: Node,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.nodeUpsertArgs['create'],
    PrismaUpdate: Prisma.nodeUpsertArgs['update'],
    PrismaModel: Prisma.nodeGetPayload<SelectWrap<Prisma.nodeSelect>>,
    PrismaSelect: Prisma.nodeSelect,
    PrismaWhere: Prisma.nodeWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            nodeEnd: 'NodeEnd',
            nodeRoutineList: 'NodeRoutineList',
            loop: 'NodeLoop',
            routineVersion: 'RoutineVersion',
        },
        prismaRelMap: {
            __typename,
            routineVersion: 'RoutineVersion',
            nodeEnd: 'NodeEnd',
            previous: 'NodeLink',
            next: 'NodeLink',
            loop: 'NodeLoop',
            nodeRoutineList: 'NodeRoutineList',
            runSteps: 'RunRoutineStep',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                columnIndex: noNull(data.columnIndex),
                rowIndex: noNull(data.rowIndex),
                type: data.type,
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'RoutineVersion', parentRelationshipName: 'nodes', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'loop', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'NodeLoop', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodeEnd', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'NodeEnd', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodeRoutineList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'NodeRoutineList', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                columnIndex: noNull(data.columnIndex),
                rowIndex: noNull(data.rowIndex),
                type: noNull(data.type),
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'nodes', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'loop', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: true, isRequired: false, objectType: 'NodeLoop', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodeEnd', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'NodeEnd', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'nodeRoutineList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'NodeRoutineList', parentRelationshipName: 'node', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
            })
        },
        yup: nodeValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: {
                private: 0,
                public: 10000,
            },
            Organization: 0,
        },
        permissionsSelect: () => ({ routineVersion: 'RoutineVersion' }),
        permissionResolvers: ({ isAdmin }) => ({
            canDelete: async () => isAdmin,
            canEdit: async () => isAdmin,
        }),
        owner: (data) => RoutineModel.validate!.owner(data.routineVersion as any),
        isDeleted: (data, languages) => RoutineModel.validate!.isDeleted(data.routineVersion as any, languages),
        isPublic: (data, languages) => RoutineModel.validate!.isPublic(data.routineVersion as any, languages),
        validations: {
            create: async ({ createMany, deltaAdding, prisma, userData }) => {
                if (createMany.length === 0) return;
                // Don't allow more than 100 nodes in a routine
                if (deltaAdding < 0) return;
                const existingCount = await prisma.routine_version.findUnique({
                    where: { id: createMany[0].routineVersionConnect },
                    include: { _count: { select: { nodes: true } } }
                });
                const totalCount = (existingCount?._count.nodes ?? 0) + deltaAdding
                if (totalCount > MAX_NODES_IN_ROUTINE) {
                    throw new CustomError('0052', 'MaxNodesReached', userData.languages, { totalCount });
                }
            }
        },
        visibility: {
            private: { routineVersion: RoutineModel.validate!.visibility.private },
            public: { routineVersion: RoutineModel.validate!.visibility.public },
            owner: (userId) => ({ routineVersion: RoutineModel.validate!.visibility.owner(userId) }),
        }
    },
})