import { Node, NodeCreateInput, NodeUpdateInput } from "../schema/types";
import { relationshipBuilderHelper } from "./builder";
import { nodeTranslationCreate, nodeTranslationUpdate, nodesCreate, nodesUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { TranslationModel } from "./translation";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { routineValidator } from "./routine";
import { cudHelper } from "./actions";
import { CustomError } from "../events";
import { organizationQuerier } from "./organization";

const MAX_NODES_IN_ROUTINE = 100;

export const nodeFormatter = (): FormatConverter<Node, any> => ({
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

export const nodeValidator = (): Validator<
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
    permissionsSelect: (userId) => ({ routineVersion: { select: routineValidator().permissionsSelect(userId) } }),
    permissionResolvers: ({ isAdmin }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
    ]),
    ownerOrMemberWhere: (userId) => ({
        routineVersion: {
            root: {
                OR: [
                    { user: { id: userId } },
                    organizationQuerier().hasRoleInOrganizationQuery(userId)
                ]
            }
        }
    }),
    owner: (data) => routineValidator().owner(data.routineVersion as any),
    isDeleted: () => false,
    isPublic: (data) => routineValidator().isPublic(data.routineVersion as any),
    validations: {
        create: async (createMany, prisma, userId, deltaAdding) => {
            if (createMany.length === 0) return;
            // Don't allow more than 100 nodes in a routine
            if (deltaAdding < 0) return;
            const existingCount = await prisma.routine_version.findUnique({
                where: { id: createMany[0].routineVersionId },
                include: { _count: { select: { nodes: true } } }
            });
            if ((existingCount?._count.nodes ?? 0) + deltaAdding > MAX_NODES_IN_ROUTINE) {
                throw new CustomError('ErrorUnknown', `To prevent performance issues, no more than ${MAX_NODES_IN_ROUTINE} nodes can be added to a routine. If you think this is a mistake, please contact us`, { trace: '0052' });
            }
        }
    }
})

export const nodeMutater = (prisma: PrismaType): Mutater<Node> => ({
    async toDBBase(userId: string, data: NodeCreateInput | NodeUpdateInput) {
        return {
            id: data.id,
            columnIndex: data.columnIndex ?? undefined,
            rowIndex: data.rowIndex ?? undefined,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: nodeTranslationCreate, update: nodeTranslationUpdate }, false),
            nodeEnd: (data as NodeCreateInput)?.nodeEndCreate ?
                this.relationshipBuilderEndNode(userId, data, true) :
                (data as NodeUpdateInput)?.nodeEndUpdate ?
                    this.relationshipBuilderEndNode(userId, data, false) :
                    undefined,
            nodeRoutineList: (data as NodeCreateInput)?.nodeRoutineListCreate ?
                await NodeRoutineListModel.mutate(prisma).relationshipBuilder!(userId, data, true) :
                (data as NodeUpdateInput)?.nodeRoutineListUpdate ?
                    await NodeRoutineListModel.mutate(prisma).relationshipBuilder!(userId, data, false) :
                    undefined,
        };
    },
    async shapeCreate(userId: string, data: NodeCreateInput): Promise<Prisma.nodeUpsertArgs['create']> {
        return {
            ...(await this.toDBBase(userId, data)),
            routineVersionId: data.routineVersionId,
            type: data.type,
        };
    },
    async shapeUpdate(userId: string, data: NodeUpdateInput): Promise<Prisma.nodeUpsertArgs['update']> {
        return this.toDBBase(userId, data);
    },
    /**
     * Add, update, or delete a node relationship on a routine
     */
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'nodes',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isTransferable: false,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
    },
    /**
     * Add, update, or remove whens from a node link
     */
    relationshipBuilderNodeLinkWhens(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'whens',
            isAdd,
            isTransferable: false,
            userId,
        })
    },
    /**
     * Add, update, or remove node link from a node orchestration
     */
    relationshipBuilderNodeLink(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'nodeLinks',
            isAdd,
            isTransferable: false,
            shape: (userId: string, cuData: { [x: string]: any }, isAdd: boolean) => {
                let { fromId, toId, ...rest } = cuData;
                return {
                    ...rest,
                    whens: this.relationshipBuilderNodeLinkWhens(userId, cuData, isAdd),
                    from: { connect: { id: cuData.fromId } },
                    to: { connect: { id: cuData.toId } },
                }
            },
            userId,
        })
    },
    /**
     * Add, update, or remove combine node data from a node.
     * Since this is a one-to-one relationship, we cannot return arrays
     */
    relationshipBuilderEndNode(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'nodeEnd',
            isAdd,
            isOneToOne: true,
            isTransferable: false,
            userId,
        })
    },
    /**
     * Add, update, or remove loop while data from a node
     */
    relationshipBuilderLoopWhiles(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'whiles',
            isAdd,
            isTransferable: false,
            userId,
        })
    },
    /**
     * Add, update, or remove loop data from a node
     */
    relationshipBuilderLoop(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'loop',
            isAdd,
            isTransferable: false,
            shape: (userId: string, cuData: { [x: string]: any }, isAdd: boolean) => ({
                ...cuData,
                whiles: this.relationshipBuilderLoopWhiles(userId, cuData, isAdd)
            }),
            userId,
        })
    },
    async cud(params: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        return cudHelper({
            ...params,
            objectType: 'Node',
            prisma,
            yup: { yupCreate: nodesCreate, yupUpdate: nodesUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
        })
    },
})

export const NodeModel = ({
    prismaObject: (prisma: PrismaType) => prisma.node,
    format: nodeFormatter(),
    mutate: nodeMutater,
    type: 'Node' as GraphQLModelType,
    validate: nodeValidator(),
})