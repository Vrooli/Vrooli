import { DeleteOneInput, Node, NodeCreateInput, NodeType, NodeUpdateInput, Success } from "../schema/types";
import { deconstructUnion, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, omitDeep, relationshipFormatter, relationshipToPrisma, RelationshipTypes, removeOwnerField, selectHelper } from "./base";
import { CustomError } from "../error";
import { CODE, condition, conditionsCreate, conditionsUpdate, nodeCreate, nodeEndCreate, nodeEndUpdate, nodeLinksCreate, nodeLinksUpdate, nodeLoopCreate, nodeLoopUpdate, nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, nodeUpdate, whilesCreate, whilesUpdate } from "@local/shared";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { node } from "@prisma/client";
import { hasProfanityRecursive } from "../utils/censor";
import { routineDBFields, RoutineModel } from "./routine";
import _ from "lodash";

const MAX_NODES_IN_ROUTINE = 100;

//==============================================================
/* #region Custom Components */
//==============================================================

type NodeFormatterType = FormatConverter<Node, node>;
/**
 * Component for formatting between graphql and prisma types
 */
export const nodeFormatter = (): NodeFormatterType => {
    return {
        dbShape: (partial: PartialSelectConvert<Node>): PartialSelectConvert<node> => {
            let modified = partial;
            console.log('in nodeFormatter.selectToDB', modified);
            // Deconstruct GraphQL unions
            modified = deconstructUnion(modified, 'data', [
                ['nodeEnd', {
                    wasSuccessful: true,
                }],
                ['nodeLoopFrom', {
                    loops: true,
                    maxLoops: true,
                    toId: true,
                    whiles: {
                        description: true,
                        title: true,
                        when: {
                            condition: true,
                        }
                    }
                }],
                ['nodeRoutineList', {
                    isOrdered: true,
                    isOptional: true,
                    routines: {
                        description: true,
                        isOptional: true,
                        title: true,
                        routineId: true,
                        routine: {
                            ...Object.fromEntries(routineDBFields.map(f => [f, true])),
                        }
                    }
                }],
                // TODO modified.nodeRedirect
            ]);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['routine', FormatterMap.Routine.dbShape],
            ]);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<node> => {
            // Convert GraphQL info into to a partial select infoect
            let modified = infoToPartialSelect(info);
            // Convert relatÎionships
            modified = relationshipFormatter(modified, [
                ['routine', FormatterMap.Routine.dbPrune],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<node> => {
            return nodeFormatter().dbShape(nodeFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Node> => {
            // Create unions
            let { nodeEnd, nodeLoopFrom, nodeRoutineList, nodeRedirect, ...modified } = obj;
            if (nodeEnd) {
                modified.data = nodeEnd;
            }
            else if (nodeLoopFrom) {
                modified.data = nodeLoopFrom;
            }
            else if (nodeRoutineList) {
                modified.data = {
                    ...nodeRoutineList,
                    routines: nodeRoutineList.routines.map((r: any) => FormatterMap.Routine.selectToGraphQL(r)),
                }
            }
            // else if (nodeRedirect) { TODO
            // }
            // Convert relationships 
            modified = relationshipFormatter(modified, [
                ['routine', FormatterMap.Routine.selectToGraphQL],
            ]);
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            return modified;
        },
    }
}

/**
 * Handles the authorized adding, updating, and deleting of nodes.
 */
const noder = (format: NodeFormatterType, prisma: PrismaType) => ({
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
     * Add, update, or remove a node relationship from a routine
     */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by nodes (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'nodes', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check if routine will pass max nodes
            this.nodeCountCheck(input.routineId, formattedInput.create.length);
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as NodeCreateInput);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as NodeUpdateInput);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async create(
        userId: string,
        input: NodeCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Check if authorized TODO
        // Check if routine will pass max nodes
        this.nodeCountCheck(input.routineId, 1);
        // Create node data
        let nodeData: { [x: string]: any } = {
            description: input.description,
            routineId: input.routineId,
            title: input.title,
            type: input.type,
        };
        // Create type-specific data
        switch (input.type) {
            case NodeType.End:
                if (!input.nodeEndCreate) throw new CustomError(CODE.InvalidArgs, 'If type is end, nodeEndCreate must be provided');
                nodeData.nodeEnd = this.relationshipBuilderEndNode(userId, input, true);
                break;
            case NodeType.Loop:
                if (!input.nodeLoopCreate) throw new CustomError(CODE.InvalidArgs, 'If type is loop, nodeLoopCreate must be provided');
                nodeData.nodeLoop = this.relationshipBuilderLoopNode(userId, input, true);
                break;
            case NodeType.RoutineList:
                if (!input.nodeRoutineListCreate) throw new CustomError(CODE.InvalidArgs, 'If type is routineList, nodeRoutineListCreate must be provided');
                nodeData.nodeRoutineList = this.relationshipBuilderRoutineListNode(userId, input, true);
                break;
        }
        // Create node
        const node = await prisma.node.create({
            data: nodeData,
            ...selectHelper(info, format.selectToDB)
        })
        // Return project
        return { ...format.selectToGraphQL(node) } as any;
    },
    async update(
        userId: string,
        input: NodeUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create node data
        let nodeData: { [x: string]: any } = {
            description: input.description,
            routineId: input.routineId,
            title: input.title,
            type: input.type,
        };
        // Create type-specific data
        switch (input.type) {
            case NodeType.End:
                if (Boolean(input.nodeEndCreate) === Boolean(input.nodeEndUpdate)) throw new CustomError(CODE.InvalidArgs, 'If type is end, nodeEndCreate xor nodeEndUpdate must be provided');
                nodeData.nodeEnd = this.relationshipBuilderEndNode(userId, input, false);
                break;
            case NodeType.Loop:
                if (Boolean(input.nodeLoopCreate) === Boolean(input.nodeLoopUpdate)) throw new CustomError(CODE.InvalidArgs, 'If type is loop, nodeLoopCreate xor nodeLoopUpdate must be provided');
                nodeData.nodeLoop = this.relationshipBuilderLoopNode(userId, input, false);
                break;
            case NodeType.RoutineList:
                if (Boolean(input.nodeRoutineListCreate) === Boolean(input.nodeRoutineListUpdate)) throw new CustomError(CODE.InvalidArgs, 'If type is routineList, nodeRoutineListCreate xor nodeRoutineListUpdate must be provided');
                nodeData.nodeRoutineList = this.relationshipBuilderRoutineListNode(userId, input, false);
                break;
        }
        // Delete old type-specific data, if needed
        // TODO
        // Find node
        let node = await prisma.node.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    { routine: { userId } }
                ]
            }
        })
        if (!node) throw new CustomError(CODE.ErrorUnknown);
        // Update node
        node = await prisma.node.update({
            where: { id: node.id },
            data: nodeData,
            ...selectHelper(info, format.selectToDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [node], info);
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Check if authorized TODO
        const isAuthorized = true;
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        await prisma.node.delete({
            where: { id: input.id }
        })
        return { success: true };
    },
    /**
     * Currently no supplemental fields
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Node>[]> {
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
    },
    profanityCheck(data: NodeCreateInput | NodeUpdateInput): void {
        if (hasProfanityRecursive(data, ['condition', 'description', 'title'])) throw new CustomError(CODE.BannedWord);
    },
    /**
     * Checks if nodes being added surpass node limit
     */
    async nodeCountCheck(
        routineId: string,
        numAdding: number = 1,
    ): Promise<void> {
        // Validate input
        if (!routineId) throw new CustomError(CODE.InvalidArgs, 'routineId must be provided in nodeCountCheck');
        // Check if routine has reached max nodes
        const nodeCount = await prisma.routine.findUnique({
            where: { id: routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if (!nodeCount) throw new CustomError(CODE.ErrorUnknown);
        if (nodeCount._count.nodes + numAdding > MAX_NODES_IN_ROUTINE) throw new CustomError(CODE.MaxNodesReached);
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function NodeModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Node;
    const format = nodeFormatter();

    return {
        prisma,
        model,
        ...format,
        ...noder(format, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================