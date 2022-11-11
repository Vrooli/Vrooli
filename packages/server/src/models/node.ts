import { Node, NodeCreateInput, NodeUpdateInput } from "../schema/types";
import { relationshipBuilderHelper, RelationshipTypes } from "./builder";
import { nodeTranslationCreate, nodeTranslationUpdate, nodesCreate, nodesUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { TranslationModel } from "./translation";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Permissioner } from "./types";
import { Prisma } from "@prisma/client";
import { routinePermissioner } from "./routine";
import { cudHelper } from "./actions";

const MAX_NODES_IN_ROUTINE = 100;

export const nodeFormatter = (): FormatConverter<Node, any> => ({
    relationshipMap: {
        '__typename': 'Node',
        'data': {
            'NodeEnd': 'NodeEnd',
            'NodeRoutineList': 'NodeRoutineList',
        },
        'loop': 'NodeLoop',
        'routine': 'Routine',
    },
})

export const nodePermissioner = (): Permissioner<any, any> => ({
    async get() {
        return [] as any;
    },
    ownershipQuery: (userId) => ({
        routineVersion: { root: routinePermissioner().ownershipQuery(userId) }
    }),
})

export const nodeValidator = () => ({
    // Don't allow more than 100 nodes in a routine
    // if (numAdding < 0) return;
    //     const existingCount = await prisma.routine_version.findUnique({
    //         where: { id: routineVersionId },
    //         include: { _count: { select: { nodes: true } } }
    //     });
    //     if ((existingCount?._count.nodes ?? 0) + numAdding > MAX_NODES_IN_ROUTINE) {
    //         throw new CustomError(CODE.ErrorUnknown, `To prevent performance issues, no more than ${MAX_NODES_IN_ROUTINE} nodes can be added to a routine. If you think this is a mistake, please contact us`, { code: genErrorCode('0052') });
    //     }
})

export const nodeMutater = (prisma: PrismaType) => ({
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
                await NodeRoutineListModel.mutate(prisma).relationshipBuilder(userId, data, true) :
                (data as NodeUpdateInput)?.nodeRoutineListUpdate ?
                    await NodeRoutineListModel.mutate(prisma).relationshipBuilder(userId, data, false) :
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
            // connect/disconnect not supported by nodes (since they can only be applied to one routine)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
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
            // connect/disconnect not supported by node links whens (since they can only be applied to one node link)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
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
            // connect/disconnect not supported by node data (since they can only be applied to one node)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
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
            // connect/disconnect not supported by node data (since they can only be applied to one node)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
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
            // connect/disconnect not supported by node data (since they can only be applied to one node)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
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
            // connect/disconnect not supported by node data (since they can only be applied to one node)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
            shape: (userId: string, cuData: { [x: string]: any }, isAdd: boolean) => ({
                ...cuData,
                whiles: this.relationshipBuilderLoopWhiles(userId, cuData, isAdd)
            }),
            userId,
        })
    },
    /**
     * Performs adds, updates, and deletes of nodes. First validates that every action is allowed.
     */
    async cud(params: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        return cudHelper({
            ...params,
            objectType: 'Node',
            prisma,
            prismaObject: (p) => p.node,
            yup: { yupCreate: nodesCreate, yupUpdate: nodesUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const NodeModel = ({
    prismaObject: (prisma: PrismaType) => prisma.node,
    format: nodeFormatter(),
    mutate: nodeMutater,
    permissions: nodePermissioner,
    type: 'Node' as GraphQLModelType,
    validate: nodeValidator(),
})