import { Node, NodeCreateInput, NodeUpdateInput } from "../schema/types";
import { relationshipToPrisma, RelationshipTypes } from "./builder";
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
        routineId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'nodes',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by nodes (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        // Further shape the input
        if (createMany) {
            let result: { [x: string]: any }[] = [];
            for (const data of createMany) {
                result.push(await this.shapeCreate(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result: { where: { [x: string]: string }, data: { [x: string]: any } }[] = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.shapeUpdate(userId, data.data as any),
                })
            }
            updateMany = result;
        }
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    /**
     * Add, update, or remove node link whens from a routine
     */
    relationshipBuilderNodeLinkWhens(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node links (since they can only be applied to one node orchestration)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'whens', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove node link from a node orchestration
     */
    relationshipBuilderNodeLink(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodeLinks', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (let data of formattedInput.create) {
                // Convert nested relationships
                data.whens = this.relationshipBuilderNodeLinkWhens(userId, data, isAdd);
                let { fromId, toId, ...rest } = data;
                data = {
                    ...rest,
                    from: { connect: { id: data.fromId } },
                    to: { connect: { id: data.toId } },
                };
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (let data of formattedInput.update) {
                // Convert nested relationships
                data.data.whens = this.relationshipBuilderNodeLinkWhens(userId, data, isAdd);
                let { fromId, toId, ...rest } = data.data;
                data.data = {
                    ...rest,
                    from: fromId ? { connect: { id: data.data.fromId } } : undefined,
                    to: toId ? { connect: { id: data.data.toId } } : undefined,
                }
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove combine node data from a node.
     * Since this is a one-to-one relationship, we cannot return arrays
     */
    relationshipBuilderEndNode(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'nodeEnd', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            formattedInput.create = formattedInput.create[0];
        }
        // Validate update
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            formattedInput.update = formattedInput.update[0].data;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop while data from a node
     */
    relationshipBuilderLoopWhiles(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'whiles', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop data from a node
     */
    relationshipBuilderLoop(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'loop', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Convert nested relationships
                data.whiles = this.relationshipBuilderLoopWhiles(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Convert nested relationships
                data.data.whiles = this.relationshipBuilderLoopWhiles(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Performs adds, updates, and deletes of nodes. First validates that every action is allowed.
     */
    async cud(params: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        return cudHelper({
            ...params,
            objectType: 'Node',
            prisma,
            prismaObject: (p) => p.node as any,
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