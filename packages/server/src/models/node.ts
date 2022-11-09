import { Count, Node, NodeCreateInput, NodeUpdateInput } from "../schema/types";
import { relationshipToPrisma, RelationshipTypes, selectHelper, modelToGraphQL, onlyValidIds } from "./builder";
import { CODE } from "@shared/consts";
import { nodeEndCreate, nodeEndUpdate, nodeLinksCreate, nodeLinksUpdate, nodeTranslationCreate, nodeTranslationUpdate, whilesCreate, whilesUpdate, whensCreate, whensUpdate, loopsCreate, loopsUpdate, nodesCreate, nodesUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { TranslationModel } from "./translation";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { CustomError, genErrorCode } from "../events";
import { FormatConverter, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";
import { routinePermissioner } from "./routine";

const MAX_NODES_IN_ROUTINE = 100;

//==============================================================
/* #region Custom Components */
//==============================================================

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

/**
 * Authorization checks
 */
export const nodeVerifier = () => ({
    /**
     * Verify that the user can modify these nodes. 
     * Also doubles as a way to verify that all nodes are in the same routine.
     * @returns Promise with routineVersionId, so we can use it later
     */
    async authorizedCheck(userId: string, nodeIds: string[], prisma: PrismaType): Promise<string | null> {
        // Find routine version id for every node
        const routineVersionIds = await prisma.node.findMany({
            where: { id: { in: nodeIds } },
            select: { routineVersionId: true },
        }).then((nodes) => nodes.map((n) => n.routineVersionId));
        // If there are no nodes, they must be new. In this case, we're all good. Return null
        if (routineVersionIds.length === 0) return null;
        // If the number of routine version ids is not equal to the number of nodes, then some nodes are new. We don't allow this.
        // We also don't allow nodes from different routines to be modified at the same time.
        if (routineVersionIds.length !== nodeIds.length || routineVersionIds.some((id) => id !== routineVersionIds[0])) {
            throw new CustomError(CODE.Unauthorized, 'Can only update nodes in the same routine', { code: genErrorCode('0050') });
        }
        // Now we should have a single routine version id. Check that the user has permission to modify it.
        const ownRoutines = await prisma.routine.findMany({
            where: {
                versions: { some: { id: routineVersionIds[0] } },
                ...routinePermissioner().ownershipQuery(userId),
            }
        })
        if (ownRoutines.length === 0) {
            throw new CustomError(CODE.Unauthorized, 'User does not have permission to modify this routine', { code: genErrorCode('0051') });
        }
        return routineVersionIds[0];
    },
    /**
     * Verify that the maximum number of nodes on a routine will not be exceeded
     */
    async maximumCheck(routineVersionId: string, numAdding: number, prisma: PrismaType): Promise<void> {
        // If removing, no need to check
        if (numAdding < 0) return;
        const existingCount = await prisma.routine_version.findUnique({
            where: { id: routineVersionId },
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
    async toDBCreate(userId: string, data: NodeCreateInput): Promise<Prisma.nodeUpsertArgs['create']> {
        return { 
            ...(await this.toDBBase(userId, data)), 
            routineVersionId: data.routineVersionId,
            type: data.type,
        };
    },
    async toDBUpdate(userId: string, data: NodeUpdateInput): Promise<Prisma.nodeUpsertArgs['update']> {
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
                result.push(await this.toDBCreate(userId, data as any));
            }
            createMany = result;
        }
        if (updateMany) {
            let result: { where: { [x: string]: string }, data: { [x: string]: any } }[] = [];
            for (const data of updateMany) {
                result.push({
                    where: data.where,
                    data: await this.toDBUpdate(userId, data.data as any),
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
        userId: string,
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
        userId: string,
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
     * NOTE: Nodes must all be applied to the same routine
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<NodeCreateInput, NodeUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        // Make sure the user has access to these nodes
        const createNodeIds: string[] = onlyValidIds(createMany?.map(node => node.id) ?? []); // Nodes can have an ID on create, unlike most other objects
        const updateNodeIds: string[] = onlyValidIds(updateMany?.map(node => node.where.id) ?? []);
        const deleteNodeIds: string[] = onlyValidIds(deleteMany ?? []);
        const allNodeIds = [...createNodeIds, ...updateNodeIds, ...deleteNodeIds];
        if (allNodeIds.length === 0) return;
        const routineVersionId: string | null = await nodeVerifier().authorizedCheck(userId, allNodeIds, prisma);
        if (createMany) {
            nodesCreate.validateSync(createMany, { abortEarly: false });
            nodeVerifier().profanityCheck(createMany);
            // Check if will pass max nodes (on routine) limit
            if (routineVersionId) await nodeVerifier().maximumCheck(routineVersionId, (createMany?.length ?? 0) - (deleteMany?.length ?? 0), prisma);
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
                    data: await this.toDBCreate(userId, input),
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
                    data: await this.toDBUpdate(userId, input.data),
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
    prismaObject: (prisma: PrismaType) => prisma.node,
    format: nodeFormatter(),
    mutate: nodeMutater,
    type: 'Node' as GraphQLModelType,
    verify: nodeVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================