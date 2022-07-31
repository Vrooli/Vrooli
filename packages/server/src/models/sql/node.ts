import { Count, Node, NodeCreateInput, NodeUpdateInput } from "../../schema/types";
import { CUDInput, CUDResult, deconstructUnion, FormatConverter, relationshipToPrisma, RelationshipTypes, selectHelper, modelToGraphQL, ValidateMutationsInput, GraphQLModelType, ModelLogic } from "./base";
import { CustomError } from "../../error";
import { CODE, nodeEndCreate, nodeEndUpdate, nodeLinksCreate, nodeLinksUpdate, nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, nodeTranslationCreate, nodeTranslationUpdate, whilesCreate, whilesUpdate, whensCreate, whensUpdate, nodeRoutineListItemTranslationCreate, nodeRoutineListItemTranslationUpdate, loopsCreate, loopsUpdate, nodesCreate, nodesUpdate } from "@local/shared";
import { PrismaType } from "types";
import { validateProfanity } from "../../utils/censor";
import { RoutineModel } from "./routine";
import { TranslationModel } from "./translation";
import { genErrorCode } from "../../logger";

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
        '__typename': GraphQLModelType.NodeRoutineList,
        'routines': {
            '__typename': GraphQLModelType.NodeRoutineListItem,
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
    async authorizedCheck(userId: string, nodeIds: string[], prisma: PrismaType): Promise<string | null> {
        //TODO
        return null;
        // let nodes = await prisma.node.findMany({
        //     where: {
        //         AND: [
        //             { id: { in: nodeIds } },
        //             {
        //                 OR: [
        //                     { routine: { userId } },
        //                     {
        //                         routine: {
        //                             organization: {
        //                                 members: {
        //                                     some: {
        //                                         userId,
        //                                         role: { in: [MemberRole.Owner, MemberRole.Admin] }
        //                                     }
        //                                 }
        //                             }
        //                         }
        //                     }
        //                 ]
        //             }
        //         ]
        //     },
        //     select: { routineId: true }
        // });
        // // Check that all nodes are in the same routine
        // const uniqueRoutineIds = new Set(nodes.map(node => node.routineId))
        // // If not routine ids found, check if all nodes are new
        // if (uniqueRoutineIds.size === 0) {
        //     const newNodes = await prisma.node.count({
        //         where: {
        //             id: { in: nodeIds },
        //         }
        //     })
        //     // If all nodes are new, return null for routineId
        //     if (newNodes === 0) return null;
        //     // If not all nodes are new, then there must be one that is not new and not owned by the user
        //     else throw new CustomError(CODE.Unauthorized, 'You do not own all nodes', { code: genErrorCode('0050') });
        // }
        // if (uniqueRoutineIds.size > 1)
        //     throw new CustomError(CODE.InvalidArgs, 'All nodes must be in the same routine!', { code: genErrorCode('0051') });
        // return uniqueRoutineIds.values().next().value;
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
            throw new CustomError(CODE.ErrorUnknown, `To prevent performance issues, no more than ${MAX_NODES_IN_ROUTINE} nodes can be added to a routine. If you think this is a mistake, please contact us`, { code: genErrorCode('0052') });
        }
    },
    profanityCheck(data: (NodeCreateInput | NodeUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => ({
            condition: d.condition,
            description: d.description,
            title: d.title,
        })));
    },
})

export const nodeMutater = (prisma: PrismaType) => ({
    async toDBShape(userId: string | null, data: NodeCreateInput | NodeUpdateInput): Promise<any> {
        let nodeData: { [x: string]: any } = {
            id: data.id,
            columnIndex: data.columnIndex ?? undefined,
            routineId: data.routineId ?? undefined,
            rowIndex: data.rowIndex ?? undefined,
            type: data.type ?? undefined,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: nodeTranslationCreate, update: nodeTranslationUpdate }, false),
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
            TranslationModel.profanityCheck(formattedInput.create);
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments
            whensUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(formattedInput.update.map(u => u.data));
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
            nodeLinksUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
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
            whilesUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
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
            loopsCreate.validateSync(formattedInput.create, { abortEarly: false });
            for (const data of formattedInput.create) {
                // Convert nested relationships
                data.whiles = this.relationshipBuilderLoopWhiles(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            loopsUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
            for (const data of formattedInput.update) {
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
        const mutate = RoutineModel.mutate(prisma);
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments
            nodeRoutineListItemsCreate.validateSync(formattedInput.create, { abortEarly: false });
            let result = [];
            for (const data of formattedInput.create) {
                result.push({
                    id: data.id,
                    index: data.index,
                    isOptional: data.isOptional,
                    routineId: await mutate.relationshipBuilder(userId, data, isAdd),
                    translations: TranslationModel.relationshipBuilder(userId, data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                })
            }
            formattedInput.create = result;
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments
            nodeRoutineListItemsUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
            let result = [];
            for (const data of formattedInput.update) {
                result.push({
                    where: data.where,
                    data: {
                        index: data.data.index ?? undefined,
                        isOptional: data.data.isOptional ?? undefined,
                        routineId: await mutate.relationshipBuilder(userId, data.data, isAdd),
                        translations: TranslationModel.relationshipBuilder(userId, data.data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
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
                id: create.id,
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
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'Must pass valid userId to validateMutations', { code: genErrorCode('0054') });
        // Make sure the user has access to these nodes
        const createNodeIds: string[] = createMany?.map(node => node.id)?.filter(id => !!id) as string[] ?? []; // Nodes can have an ID on create, unlike most other objects
        const updateNodeIds: string[] = updateMany?.map(node => node.where.id)?.filter(id => !!id) as string[] ?? [];
        const deleteNodeIds: string[] = deleteMany?.filter(Boolean) ?? [];
        const allNodeIds = [...createNodeIds, ...updateNodeIds, ...deleteNodeIds];
        if (allNodeIds.length === 0) return;
        const routineId: string | null = await nodeVerifier().authorizedCheck(userId, allNodeIds, prisma);
        if (createMany) {
            nodesCreate.validateSync(createMany, { abortEarly: false });
            nodeVerifier().profanityCheck(createMany);
            // Check if will pass max nodes (on routine) limit
            if (routineId) await nodeVerifier().maximumCheck(routineId, (createMany?.length ?? 0) - (deleteMany?.length ?? 0), prisma);
        }
        if (updateMany) {
            nodesUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            nodeVerifier().profanityCheck(updateMany.map(u => u.data));
        }
    },
    /**
     * Performs adds, updates, and deletes of nodes. First validates that every action is allowed.
     */
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<NodeCreateInput, NodeUpdateInput>): Promise<CUDResult<Node>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Create object
                const currCreated = await prisma.node.create({
                    data: await this.toDBShape(userId, input),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
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
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
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

export const NodeModel = ({
    prismaObject: (prisma: any) => prisma.node,
    format: nodeFormatter(),
    mutate: nodeMutater,
    verify: nodeVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================