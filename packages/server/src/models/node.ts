import { Count, Node, NodeCreateInput, NodeType, NodeUpdateInput } from "../schema/types";
import { CUDInput, CUDResult, deconstructUnion, FormatConverter, relationshipToPrisma, RelationshipTypes, selectHelper, modelToGraphQL, ValidateMutationsInput, GraphQLModelType } from "./base";
import { CustomError } from "../error";
import { CODE, nodeCreate, nodeEndCreate, nodeEndUpdate, nodeLinksCreate, nodeLinksUpdate, loopCreate, loopUpdate, nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, nodeTranslationCreate, nodeTranslationUpdate, nodeUpdate, whilesCreate, whilesUpdate, whensCreate, whensUpdate, nodeRoutineListItemTranslationCreate, nodeRoutineListItemTranslationUpdate } from "@local/shared";
import { PrismaType } from "types";
import { hasProfanityRecursive } from "../utils/censor";
import { RoutineModel } from "./routine";
import pkg from '@prisma/client';
import { TranslationModel } from "./translation";
const { MemberRole } = pkg;

const MAX_NODES_IN_ROUTINE = 100;

//==============================================================
/* #region Custom Components */
//==============================================================

export const nodeFormatter = (): FormatConverter<Node> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Node,
        'data': {
            'NodeEnd': GraphQLModelType.NodeEnd,
            'NodeRoutineList': GraphQLModelType.NodeRoutineList,
        },
        'loop': GraphQLModelType.NodeLoop,
        'routine': GraphQLModelType.Routine,
    },
    constructUnions: (data) => {
        let { nodeEnd, nodeRoutineList, ...modified } = data;
        modified.data = nodeEnd ?? nodeRoutineList;
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = deconstructUnion(partial, 'data',
            [
                [GraphQLModelType.NodeEnd, 'nodeEnd'],
                [GraphQLModelType.NodeRoutineList, 'nodeRoutineList'],
            ]);
        return modified;
    },
})

export const nodeRoutineListFormatter = (): FormatConverter<Node> => ({
    relationshipMap: {
        'routines': {
            '__typename': GraphQLModelType.NodeRoutineList,
            'routine': GraphQLModelType.Routine,
        },
    },
})

/**
 * Authorization checks
 */
export const nodeVerifier = () => ({
    /**
     * Verify that the user can modify these nodes. 
     * Also doubles as a way to verify that all nodes are in the same routine.
     * @returns Promise with routineId, so we can use it later
     */
    async authorizedCheck(userId: string, nodeIds: string[], prisma: PrismaType): Promise<string> {
        let nodes = await prisma.node.findMany({
            where: {
                AND: [
                    { id: { in: nodeIds } },
                    {
                        OR: [
                            { routine: { userId } },
                            {
                                routine: {
                                    organization: {
                                        members: {
                                            some: {
                                                userId,
                                                role: { in: [MemberRole.Owner, MemberRole.Admin] }
                                            }
                                        }
                                    }
                                }
                            }
                        ]
                    }
                ]
            },
            select: { routineId: true }
        });
        // Check that all nodes are in the same routine
        const uniqueRoutineIds = new Set(nodes.map(node => node.routineId))
        if (uniqueRoutineIds.size === 0) throw new CustomError(CODE.Unauthorized, 'You do not own all nodes');
        if (uniqueRoutineIds.size > 1) throw new CustomError(CODE.InvalidArgs, 'All nodes must be in the same routine!');
        return uniqueRoutineIds.values().next().value;
    },
    /**
     * Verify that the maximum number of nodes on a routine will not be exceeded
     */
    async maximumCheck(routineId: string, numAdding: number, prisma: PrismaType): Promise<void> {
        // If removing, no need to check
        if (numAdding < 0) return;
        const existingCount = await prisma.routine.findUnique({
            where: { id: routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if ((existingCount?._count.nodes ?? 0) + numAdding > MAX_NODES_IN_ROUTINE) {
            throw new CustomError(CODE.ErrorUnknown, `To prevent performance issues, no more than ${MAX_NODES_IN_ROUTINE} nodes can be added to a routine. If you think this is a mistake, please contact us`);
        }
    },
    profanityCheck(data: NodeCreateInput | NodeUpdateInput): void {
        if (hasProfanityRecursive(data, ['condition', 'description', 'title'])) throw new CustomError(CODE.BannedWord);
    },
})

export const nodeMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShape(userId: string | null, data: NodeCreateInput | NodeUpdateInput): Promise<any> {
        let nodeData: { [x: string]: any } = {
            id: data.id,
            columnIndex: data.columnIndex,
            routineId: data.routineId,
            rowIndex: data.rowIndex,
            type: data.type,
            translations: TranslationModel().relationshipBuilder(userId, data, { create: nodeTranslationCreate, update: nodeTranslationUpdate }, false),
        };
        // Create type-specific data, and make sure other types are null
        nodeData.nodeEnd = undefined;
        nodeData.nodeRoutineList = undefined;
        if ((data as NodeCreateInput)?.nodeEndCreate) nodeData.nodeEnd = this.relationshipBuilderEndNode(userId, data, true);
        else if ((data as NodeUpdateInput)?.nodeEndUpdate) nodeData.nodeEnd = this.relationshipBuilderEndNode(userId, data, false);
        if ((data as NodeCreateInput).nodeRoutineListCreate) nodeData.nodeRoutineList = await this.relationshipBuilderRoutineListNode(userId, data, true);
        else if ((data as NodeUpdateInput)?.nodeRoutineListUpdate) nodeData.nodeRoutineList = await this.relationshipBuilderRoutineListNode(userId, data, false);
        if (nodeData.loop) {
            if (data.loopCreate) nodeData.loop = this.relationshipBuilderLoop(userId, data, true);
            else if ((data as NodeUpdateInput)?.loopUpdate) nodeData.loop = this.relationshipBuilderLoop(userId, data, false);
        }
        return nodeData;
    },
    /**
     * Add, update, or delete a node relationship on a routine
     */
    async relationshipBuilder(
        userId: string | null,
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
            let result = [];
            for (const data of createMany) {
                result.push(await this.toDBShape(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.toDBShape(userId, data.data as any),
                })
            }
            updateMany = result;
        }
        // Validate input, with routine ID added to each update node
        await this.validateMutations({
            userId,
            createMany: createMany as NodeCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: NodeUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
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
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node links (since they can only be applied to one node orchestration)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'whens', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments
            whensCreate.validateSync(formattedInput.create, { abortEarly: false });
            for (const data of formattedInput.create) {
                TranslationModel().profanityCheck(data);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments
            whensUpdate.validateSync(formattedInput.update, { abortEarly: false });
            for (const data of formattedInput.update) {
                TranslationModel().profanityCheck(data);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove node link from a node orchestration
     */
    relationshipBuilderNodeLink(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodeLinks', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments
            nodeLinksCreate.validateSync(formattedInput.create, { abortEarly: false });
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
            // Check for valid arguments
            nodeLinksUpdate.validateSync(formattedInput.update, { abortEarly: false });
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
        userId: string | null,
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
            // Check for valid arguments
            nodeEndCreate.validateSync(formattedInput.create, { abortEarly: false });
        }
        // Validate update
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            formattedInput.update = formattedInput.update[0].data;
            // Check for valid arguments
            nodeEndUpdate.validateSync(formattedInput.update, { abortEarly: false });
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop while data from a node
     */
    relationshipBuilderLoopWhiles(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'whiles', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments (censored words must be checked in earlier function)
            whilesCreate.validateSync(formattedInput.create, { abortEarly: false });
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments (censored words must be checked in earlier function)
            whilesUpdate.validateSync(formattedInput.update, { abortEarly: false });
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop data from a node
     */
    relationshipBuilderLoop(
        userId: string | null,
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
                // Check for valid arguments
                loopCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.whiles = this.relationshipBuilderLoopWhiles(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                loopUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.data.whiles = this.relationshipBuilderLoopWhiles(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine list node item data from a node
     */
    async relationshipBuilderRoutineListNodeItem(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'routines', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        const routineModel = RoutineModel(prisma);
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments
            nodeRoutineListItemsCreate.validateSync(formattedInput.create, { abortEarly: false });
            let result = [];
            for (const data of formattedInput.create) {
                // Routines are a one-to-one relationship, so we have to get rid of the array
                let routineRel = await routineModel.relationshipBuilder(userId, data, isAdd, 'routine')
                if (routineRel?.connect) routineRel.connect = routineRel.connect[0];
                result.push({
                    index: data.index,
                    isOptional: data.isOptional,
                    routine: routineRel,
                    translations: TranslationModel().relationshipBuilder(userId, data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                })
            }
            formattedInput.create = result;
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments
            nodeRoutineListItemsUpdate.validateSync(formattedInput.update, { abortEarly: false });
            let result = [];
            for (const data of formattedInput.update) {
                // Cannot switch routines, so no need to worry about it in the update
                result.push({
                    where: data.where,
                    data: {
                        index: data.data.index,
                        isOptional: data.data.isOptional,
                        translations: TranslationModel().relationshipBuilder(userId, data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                    }
                })
            }
            formattedInput.update = result;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine list node data from a node.
     * Since this is a one-to-one relationship, we cannot return arrays
     */
    async relationshipBuilderRoutineListNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'nodeRoutineList', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            const create = formattedInput.create[0];
            // Check for valid arguments
            nodeRoutineListCreate.validateSync(create, { abortEarly: false });
            // Convert nested relationships
            formattedInput.create = {
                isOrdered: create.isOrdered,
                isOptional: create.isOptional,
                routines: await this.relationshipBuilderRoutineListNodeItem(userId, create, isAdd)
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = formattedInput.update[0].data;
            // Check for valid arguments
            nodeRoutineListUpdate.validateSync(update, { abortEarly: false });
            // Convert nested relationships
            formattedInput.update = {
                isOrdered: update.isOrdered,
                isOptional: update.isOptional,
                routines: await this.relationshipBuilderRoutineListNodeItem(userId, update, isAdd)
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * NOTE: Nodes must all be applied to the same routine
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<NodeCreateInput, NodeUpdateInput>): Promise<void> {
        if (!userId) throw new CustomError(CODE.Unauthorized);
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        // Make sure the user has access to these nodes
        const createNodeIds = createMany?.map(node => node.id) ?? []; // Nodes can have an ID on create, unlike most other objects
        const updateNodeIds = updateMany?.map(node => node.where.id) ?? [];
        const deleteNodeIds = deleteMany ?? [];
        const allNodeIds = [...createNodeIds, ...updateNodeIds, ...deleteNodeIds];
        if (allNodeIds.length === 0) return;
        const routineId = await verifier.authorizedCheck(userId, allNodeIds, prisma);
        if (createMany) {
            createMany.forEach(input => nodeCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check if will pass max nodes (on routine) limit
            await verifier.maximumCheck(routineId, (createMany?.length ?? 0) - (deleteMany?.length ?? 0), prisma);
        }
        if (updateMany) {
            updateMany.forEach(input => nodeUpdate.validateSync(input.data, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input.data));
        }
    },
    /**
     * Performs adds, updates, and deletes of nodes. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Create object
                const currCreated = await prisma.node.create({
                    data: await this.toDBShape(userId, input),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Update object
                const currUpdated = await prisma.node.update({
                    where: input.where,
                    data: await this.toDBShape(userId, input.data),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.node.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function NodeModel(prisma: PrismaType) {
    const prismaObject = prisma.node;
    const format = nodeFormatter();
    const verify = nodeVerifier();
    const mutate = nodeMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        ...format,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================