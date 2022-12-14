import { Node, NodeCreateInput, NodeUpdateInput, SessionUser } from "../endpoints/types";
import { nodesCreate, nodesUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { Formatter, Validator, Mutater, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { CustomError } from "../events";
import { RoutineModel } from "./routine";
import { bestLabel, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { noNull, shapeHelper } from "../builders";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeCreateInput,
    GqlUpdate: NodeUpdateInput,
    GqlModel: Node,
    GqlPermission: any,
    PrismaCreate: Prisma.nodeUpsertArgs['create'],
    PrismaUpdate: Prisma.nodeUpsertArgs['update'],
    PrismaModel: Prisma.nodeGetPayload<SelectWrap<Prisma.nodeSelect>>,
    PrismaSelect: Prisma.nodeSelect,
    PrismaWhere: Prisma.nodeWhereInput,
}

const __typename = 'Node' as const;
const MAX_NODES_IN_ROUTINE = 100;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        data: {
            nodeEnd: 'NodeEnd',
            nodeRoutineList: 'NodeRoutineList',
        },
        loop: 'NodeLoop',
        routineVersion: 'RoutineVersion',
    },
})

const validator = (): Validator<Model> => ({
    validateMap: {
        __typename: 'Node',
        routineVersion: 'Routine',
    },
    isTransferable: false,
    maxObjects: {
        User: {
            private: 0,
            public: 10000,
        },
        Organization: 0,
    },
    permissionsSelect: () => ({}) as any,
    //permissionsSelect: (...params) => ({ routineVersion: { select: RoutineModel.validate.permissionsSelect(...params) } }),
    permissionResolvers: ({ isAdmin }) => ({
        canDelete: async () => isAdmin,
        canEdit: async () => isAdmin,
    }),
    owner: (data) => RoutineModel.validate.owner(data.routineVersion as any),
    isDeleted: () => false,
    isPublic: (data, languages) => RoutineModel.validate.isPublic(data.routineVersion as any, languages),
    validations: {
        create: async ({ createMany, deltaAdding, prisma, userData }) => {
            if (createMany.length === 0) return;
            // Don't allow more than 100 nodes in a routine
            if (deltaAdding < 0) return;
            const existingCount = await prisma.routine_version.findUnique({
                where: { id: createMany[0].routineVersionId },
                include: { _count: { select: { nodes: true } } }
            });
            const totalCount = (existingCount?._count.nodes ?? 0) + deltaAdding
            if (totalCount > MAX_NODES_IN_ROUTINE) {
                throw new CustomError('0052', 'MaxNodesReached', userData.languages, { totalCount });
            }
        }
    },
    visibility: {
        private: { routineVersion: RoutineModel.validate.visibility.private },
        public: { routineVersion: RoutineModel.validate.visibility.public },
        owner: (userId) => ({ routineVersion: RoutineModel.validate.visibility.owner(userId) }),
    }
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => ({
            id: data.id,
            columnIndex: noNull(data.columnIndex),
            rowIndex: noNull(data.rowIndex),
            type: data.type,
            routineVersionId: data.routineVersionId,
            ...(await shapeHelper({ relation: 'loop', relTypes: ['Create'], isOneToOne: true, isRequired: false,  objectType: 'NodeLoop', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'nodeEnd', relTypes: ['Create'], isOneToOne: true, isRequired: false,  objectType: 'NodeEnd', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'nodeRoutineList', relTypes: ['Create'], isOneToOne: true, isRequired: false,  objectType: 'NodeRoutineList', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
        }),
        update: async ({ data, prisma, userData }) => ({
            id: data.id,
            columnIndex: noNull(data.columnIndex),
            rowIndex: noNull(data.rowIndex),
            type: noNull(data.type),
            routineVersionId: noNull(data.routineVersionId),
            ...(await shapeHelper({ relation: 'loop', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: true, isRequired: false, objectType: 'NodeLoop', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'nodeEnd', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'NodeEnd', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'nodeRoutineList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'NodeRoutineList', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
        })
    },
    yup: { create: nodesCreate, update: nodesUpdate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const NodeModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    validate: validator(),
})