import { Count, Node, NodeCreateInput, NodeType, NodeUpdateInput } from "../schema/types";
import { CUDInput, CUDResult, deconstructUnion, FormatConverter, relationshipToPrisma, RelationshipTypes, selectHelper, modelToGraphQL, ValidateMutationsInput, GraphQLModelType } from "./base";
import { CustomError } from "../error";
import { CODE, condition, conditionsCreate, conditionsUpdate, nodeCreate, nodeEndCreate, nodeEndUpdate, nodeLinksCreate, nodeLinksUpdate, nodeLoopCreate, nodeLoopUpdate, nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, nodeUpdate, whilesCreate, whilesUpdate } from "@local/shared";
import { PrismaType } from "types";
import { hasProfanityRecursive } from "../utils/censor";
import { RoutineModel } from "./routine";
import pkg from '@prisma/client';
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
            'NodeLoop': GraphQLModelType.NodeLoop,
            'NodeRoutineList': GraphQLModelType.NodeRoutineList,
        },
        'routine': GraphQLModelType.Routine,
    },
    constructUnions: (data) => {
        let { nodeEnd, nodeLoopFrom, nodeRoutineList, nodeRedirect, ...modified } = data;
        if (nodeEnd) {
            modified.data = nodeEnd;
        }
        else if (nodeLoopFrom) {
            modified.data = nodeLoopFrom;
        }
        else if (nodeRoutineList) {
            modified.data = nodeRoutineList;
        }
        // else if (nodeRedirect) { TODO
        // }
        return modified;
    },
    deconstructUnions: (partial) => {
        console.log('in node deconstructunions')
        let modified = deconstructUnion(partial, 'data', 
        [
            [GraphQLModelType.NodeEnd, 'nodeEnd'],
            [GraphQLModelType.NodeLoop, 'nodeLoopFrom'],
            [GraphQLModelType.NodeRoutineList, 'nodeRoutineList'],
            // TODO modified.nodeRedirect
        ]);
        return modified;
    },
})

/**
 * Authorization checks
 */
export const nodeVerifier = () => ({
    /**
     * Verify that the user can modify the routine of the node(s)
     */
    async authorizedCheck(userId: string, routineId: string, prisma: PrismaType): Promise<void> {
        let routine = await prisma.routine.findFirst({
            where: {
                AND: [
                    { id: routineId },
                    {
                        OR: [
                            { userId },
                            {
                                organization: {
                                    members: {
                                        some: {
                                            userId,
                                            role: { in: [MemberRole.Owner, MemberRole.Admin] }
                                        }
                                    }
                                }
                            }
                        ]
                    }
                ]
            }
        })
        if (!routine) throw new CustomError(CODE.Unauthorized, 'User does not own the routine, or is not an admin of its organization');
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
    /**
     * Add, update, or remove a node relationship from a routine
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by nodes (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodes', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate input
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({ 
            userId, 
            createMany: createMany as NodeCreateInput[], 
            updateMany: updateMany as NodeUpdateInput[], 
            deleteMany: deleteMany?.map(d => d.id) 
        });
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove node link condition case from a routine
     */
    relationshipBuilderNodeLinkConditionCase(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node links (since they can only be applied to one node orchestration)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'when', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (let data of formattedInput.create) {
                // Check for valid arguments (censored words must be checked in earlier function)
                condition.validateSync(data.condition, { abortEarly: false });
                // Convert nested relationships
                data = { create: { condition: data.condition } }
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (let data of formattedInput.update) {
                // Check for valid arguments (censored words must be checked in earlier function)
                condition.validateSync(data.condition, { abortEarly: false });
                // Convert nested relationships
                data = { update: { condition: data.condition } }
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove node link condition from a routine
     */
    relationshipBuilderNodeLinkCondition(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node links (since they can only be applied to one node orchestration)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'conditions', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments (censored words must be checked in earlier function)
                conditionsCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.when = this.relationshipBuilderNodeLinkConditionCase(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments (censored words must be checked in earlier function)
                conditionsUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.when = this.relationshipBuilderNodeLinkConditionCase(userId, data, isAdd);
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
            for (let data of formattedInput.create) {
                // Check for valid arguments
                nodeLinksCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.decisions = this.relationshipBuilderNodeLinkCondition(userId, data, isAdd);
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
                // Check for valid arguments
                nodeLinksUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.decisions = this.relationshipBuilderNodeLinkCondition(userId, data, isAdd);
                let { fromId, toId, ...rest } = data;
                data = {
                    ...rest,
                    from: fromId ? { connect: { id: data.fromId } } : undefined,
                    to: toId ? { connect: { id: data.toId } } : undefined,
                }
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove combine node data from a node
     */
    relationshipBuilderEndNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodeEnd', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeEndCreate.validateSync(data, { abortEarly: false });
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeEndUpdate.validateSync(data, { abortEarly: false });
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop node while case data from a node
     */
    relationshipBuilderLoopNodeWhileCase(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'when', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (let data of formattedInput.create) {
                // Check for valid arguments (censored words must be checked in earlier function)
                condition.validateSync(data.condition, { abortEarly: false });
                // Convert nested relationships
                data = { create: { condition: data.condition } }
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (let data of formattedInput.update) {
                // Check for valid arguments (censored words must be checked in earlier function)
                condition.validateSync(data.condition, { abortEarly: false });
                // Convert nested relationships
                data = { update: { condition: data.condition } }
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop node data from a node
     */
    relationshipBuilderLoopNodeWhile(
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
            for (const data of formattedInput.create) {
                // Check for valid arguments (censored words must be checked in earlier function)
                whilesCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.when = this.relationshipBuilderLoopNodeWhileCase(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                whilesUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.when = this.relationshipBuilderLoopNodeWhileCase(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove loop node data from a node
     */
    relationshipBuilderLoopNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodeLoop', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeLoopCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.whiles = this.relationshipBuilderLoopNodeWhile(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeLoopUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.whiles = this.relationshipBuilderLoopNodeWhile(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine list node item data from a node
     */
    relationshipBuilderRoutineListNodeItem(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'routines', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        const routineModel = RoutineModel(prisma);
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeRoutineListItemsCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.routines = routineModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeRoutineListItemsUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.routines = routineModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine list node data from a node
     */
    relationshipBuilderRoutineListNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodeRoutineList', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeRoutineListCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.routines = this.relationshipBuilderRoutineListNodeItem(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeRoutineListUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.routines = this.relationshipBuilderRoutineListNodeItem(userId, data, isAdd);
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
        // Make sure every node applies to the same routine
        const routineId = Array.isArray(createMany) && createMany.length > 0
            ? createMany[0].routineId
            : Array.isArray(updateMany) && updateMany.length > 0
                ? updateMany[0].routineId
                : Array.isArray(deleteMany) && deleteMany.length > 0
                    ? deleteMany[0] : null;
        if (!routineId ||
            createMany?.some(n => n.routineId !== routineId) ||
            updateMany?.some(n => n.routineId !== routineId ||
                deleteMany?.some(n => n !== routineId))) throw new CustomError(CODE.InvalidArgs, 'All nodes must be in the same routine!');
        // Make sure the user has access to the routine
        await verifier.authorizedCheck(userId, routineId, prisma);
        if (createMany) {
            createMany.forEach(input => nodeCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check if will pass max nodes (on routine) limit
            await verifier.maxNodesCheck(routineId, (createMany?.length ?? 0) - (deleteMany?.length ?? 0), prisma);
        }
        if (updateMany) {
            updateMany.forEach(input => nodeUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
        }
    },
    /**
     * Performs adds, updates, and deletes of nodes. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        /**
         * Helper function for creating create/update Prisma value
         */
        const createData = async (input: NodeCreateInput | NodeUpdateInput): Promise<{ [x: string]: any }> => {
            let nodeData: { [x: string]: any } = {
                description: input.description,
                routineId: input.routineId,
                title: input.title,
                type: input.type,
            };
            // Create type-specific data, and make sure other types are null
            nodeData.nodeEnd = null;
            nodeData.nodeLoopFrom = null;
            nodeData.nodeRoutineList = null;
            switch (input.type) {
                case NodeType.End:
                    if (!input.nodeEndCreate) throw new CustomError(CODE.InvalidArgs, 'If type is end, nodeEndCreate must be provided');
                    nodeData.nodeEnd = this.relationshipBuilderEndNode(userId, input, true);
                    break;
                case NodeType.Loop:
                    if (!input.nodeLoopCreate) throw new CustomError(CODE.InvalidArgs, 'If type is loop, nodeLoopCreate must be provided');
                    nodeData.nodeLoopFrom = this.relationshipBuilderLoopNode(userId, input, true);
                    break;
                case NodeType.RoutineList:
                    if (!input.nodeRoutineListCreate) throw new CustomError(CODE.InvalidArgs, 'If type is routineList, nodeRoutineListCreate must be provided');
                    nodeData.nodeRoutineList = this.relationshipBuilderRoutineListNode(userId, input, true);
                    break;
            }
            return nodeData;
        }
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await createData(input);
                // Create object
                const currCreated = await prisma.node.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                const data = await createData(input);
                // Update object
                const currUpdated = await prisma.node.update({
                    where: { id: input.id },
                    data,
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