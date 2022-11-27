import { Node, NodeCreateInput, NodeUpdateInput, SessionUser } from "../schema/types";
import { nodesCreate, nodesUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { CustomError } from "../events";
import { RoutineModel } from "./routine";
import { OrganizationModel } from "./organization";
import { relBuilderHelper } from "../actions";
import { translationRelationshipBuilder } from "../utils";

const MAX_NODES_IN_ROUTINE = 100;

const formatter = (): Formatter<Node, any> => ({
    relationshipMap: {
        __typename: 'Node',
        data: {
            NodeEnd: 'NodeEnd',
            NodeRoutineList: 'NodeRoutineList',
        },
        loop: 'NodeLoop',
        routine: 'Routine',
    },
})

const validator = (): Validator<
    NodeCreateInput,
    NodeUpdateInput,
    Node,
    Prisma.nodeGetPayload<{ select: { [K in keyof Required<Prisma.nodeSelect>]: true } }>,
    any,
    Prisma.nodeSelect,
    Prisma.nodeWhereInput
> => ({
    validateMap: {
        __typename: 'Node',
        routineVersion: 'Routine',
    },
    isTransferable: false,
    permissionsSelect: (...params) => ({ routineVersion: { select: RoutineModel.validate.permissionsSelect(...params) } }),
    permissionResolvers: ({ isAdmin }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
    ]),
    ownerOrMemberWhere: (userId) => ({
        routineVersion: {
            root: {
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }
        }
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
    }
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: NodeCreateInput | NodeUpdateInput, isAdd: boolean) => {
    // Make sure there isn't both end node and routine list node data
    const result = { nodeEnd: null, nodeRoutineList: null }
    return {
        ...result,
        id: data.id,
        columnIndex: data.columnIndex ?? undefined,
        rowIndex: data.rowIndex ?? undefined,
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
        nodeEnd: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'nodeEnd', objectType: 'NodeEnd', prisma, userData }),
        nodeRoutineList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'nodeRoutineList', objectType: 'NodeRoutineList', prisma, userData }),
        // loop: asdfa
    }
}

const mutater = (): Mutater<
    Node,
    { graphql: NodeCreateInput, db: Prisma.nodeUpsertArgs['create'] },
    { graphql: NodeUpdateInput, db: Prisma.nodeUpsertArgs['update'] },
    { graphql: NodeCreateInput, db: Prisma.nodeCreateWithoutRoutineVersionInput },
    { graphql: NodeUpdateInput, db: Prisma.nodeUpdateWithoutRoutineVersionInput }
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...(await shapeBase(prisma, userData, data, true)),
                permissions: JSON.stringify({}), //TODO
                routineVersionId: data.routineVersionId,
                type: data.type,
            };
        },
        update: async ({ data, prisma, userData }) => {
            return await shapeBase(prisma, userData, data, false);
        },
        relCreate: mutater().shape.create,
        relUpdate: mutater().shape.update,
    },
    yup: { create: nodesCreate, update: nodesUpdate },
})

export const NodeModel = ({
    delegate: (prisma: PrismaType) => prisma.node,
    format: formatter(),
    mutate: mutater(),
    type: 'Node' as GraphQLModelType,
    validate: validator(),
})