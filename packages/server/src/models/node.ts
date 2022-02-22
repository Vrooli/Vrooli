import { DeleteOneInput, Node, NodeCreateInput, NodeUpdateInput, Success } from "../schema/types";
import { addOwnerField, FormatConverter, InfoType, MODEL_TYPES, relationshipToPrisma, removeOwnerField, selectHelper } from "./base";
import { CustomError } from "../error";
import { CODE, condition, decisionsCreate, decisionsUpdate, MemberRole, nodeCombineCreate, nodeCombineUpdate, nodeCreate, nodeDecisionCreate, nodeDecisionUpdate, nodeEndCreate, nodeEndUpdate, nodeLoopCreate, nodeLoopUpdate, nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, NodeType, nodeUpdate, whilesCreate, whilesUpdate } from "@local/shared";
import { PrismaType, RecursivePartial } from "types";
import { node } from "@prisma/client";
import { hasProfanity, hasProfanityRecursive } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { RoutineModel } from "./routine";

const MAX_NODES_IN_ROUTINE = 100;

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
export const nodeFormatter = (): FormatConverter<Node, node> => {
    return {
        toDB: (obj: RecursivePartial<Node>): RecursivePartial<node> => {
            let modified: any = obj;
            // Add owner fields to routine, so we can calculate user's role later
            modified.routine = removeOwnerField(modified.routine ?? {});
            // // Convert data field to select all data types TODO
            // if (modified.data) {
            //     modified.nodeCombine = {
            //         from: {
            //             fromId: true
            //         },
            //     }
            //     modified.nodeDecision = {

            //     }
            //     modified.nodeEnd = {

            //     }
            //     modified.nodeLoop = {

            //     }
            //     modified.nodeRoutineList = {

            //     }
            //     modified.nodeRedirect = {

            //     }
            //     modified.nodeStart = {

            //     }
            //     delete modified.data;
            // }
            // Remove calculated fields
            delete modified.role;
            return modified
        },
        toGraphQL: (obj: RecursivePartial<node>): RecursivePartial<Node> => {
            // Create data field from data types
            let modified: any = obj;
            // Add owner fields to routine, so we can calculate user's role later
            modified.routine = addOwnerField(modified.routine ?? {});
            // if (obj.nodeCombine) { TODO
            // }
            // else if (obj.nodeDecision) {
            // }
            // else if (obj.nodeEnd) {
            // }
            // else if (obj.nodeLoop) {
            // }
            // else if (obj.nodeRoutineList) {
            // }
            // else if (obj.nodeRedirect) {
            // }
            // else if (obj.nodeStart) {
            // }
            delete modified.nodeCombine;
            delete modified.nodeDecision;
            delete modified.nodeEnd;
            delete modified.nodeLoop;
            delete modified.nodeRoutineList;
            delete modified.nodeRedirect;
            delete modified.nodeStart;
            return modified;
        },
    }
}

/**
 * Handles the authorized adding, updating, and deleting of nodes.
 */
const noder = (format: FormatConverter<Node, node>, prisma: PrismaType) => ({
    /**
     * Add, update, or remove combine node data from a node
     */
    relationshipBuilderCombineNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma(input, 'nodeCombine', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeCombineCreate.validateSync(data, { abortEarly: false });
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeCombineUpdate.validateSync(data, { abortEarly: false });
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove decision node item whens data from a node
     */
    relationshipBuilderDecisionNodeItemCase(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma(input, 'when', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (let data of formattedInput.create) {
                // Check for valid arguments
                condition.validateSync(data.condition, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.condition)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data = { create: { condition: data.condition } }
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (let data of formattedInput.update) {
                // Check for valid arguments
                decisionsUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.condition)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data = { update: { condition: data.condition } }
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove decision node items data from a node
     */
    relationshipBuilderDecisionNodeItem(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma(input, 'decisions', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                decisionsCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data.when = this.relationshipBuilderDecisionNodeItemCase(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                decisionsUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data.when = this.relationshipBuilderDecisionNodeItemCase(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove decision node data from a node
     */
    relationshipBuilderDecisionNode(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma(input, 'nodeDecision', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                nodeDecisionCreate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.decisions = this.relationshipBuilderDecisionNodeItem(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                nodeDecisionUpdate.validateSync(data, { abortEarly: false });
                // Convert nested relationships
                data.decisions = this.relationshipBuilderDecisionNodeItem(userId, data, isAdd);
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
        let formattedInput = relationshipToPrisma(input, 'nodeEnd', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
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
        let formattedInput = relationshipToPrisma(input, 'when', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (let data of formattedInput.create) {
                // Check for valid arguments
                condition.validateSync(data.condition, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.condition)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data = { create: { condition: data.condition } }
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (let data of formattedInput.update) {
                // Check for valid arguments
                decisionsUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.condition)) throw new CustomError(CODE.BannedWord);
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
        let formattedInput = relationshipToPrisma(input, 'whiles', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                whilesCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data.when = this.relationshipBuilderLoopNodeWhileCase(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                whilesUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
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
        let formattedInput = relationshipToPrisma(input, 'nodeLoop', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
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
        let formattedInput = relationshipToPrisma(input, 'routines', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
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
        let formattedInput = relationshipToPrisma(input, 'nodeRoutineList', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
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
    async create(
        userId: string,
        input: NodeCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanityRecursive(input, ['condition', 'description', 'title'])) throw new CustomError(CODE.BannedWord);
        // Check if authorized TODO
        // Check if routine has reached max nodes
        const nodeCount = await prisma.routine.findUnique({
            where: { id: input.routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if (!nodeCount) throw new CustomError(CODE.ErrorUnknown);
        if (nodeCount._count.nodes >= MAX_NODES_IN_ROUTINE) throw new CustomError(CODE.MaxNodesReached);
        // Create node data
        let nodeData: { [x: string]: any } = {
            description: input.description,
            nextId: input.nextId,
            previousId: input.previousId,
            routineId: input.routineId,
            title: input.title,
            type: input.type,
        };
        // Create type-specific data
        switch (input.type) {
            case NodeType.Combine:
                if (!input.nodeCombineCreate) throw new CustomError(CODE.InvalidArgs, 'If type is combine, nodeCombineCreate must be provided');
                nodeData.nodeCombine = this.relationshipBuilderCombineNode(userId, input, true);
                break;
            case NodeType.Decision:
                if (!input.nodeDecisionCreate) throw new CustomError(CODE.InvalidArgs, 'If type is decision, nodeDecisionCreate must be provided');
                nodeData.nodeDecision = this.relationshipBuilderDecisionNode(userId, input, true);
                break;
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
            ...selectHelper<Node, node>(info, format.toDB)
        })
        // Return project with "role" field
        return { ...format.toGraphQL(node), role: MemberRole.Owner } as any;
    },
    async update(
        userId: string,
        input: NodeUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanityRecursive(input, ['condition', 'description', 'title'])) throw new CustomError(CODE.BannedWord);
        // Create node data
        let nodeData: { [x: string]: any } = {
            description: input.description,
            nextId: input.nextId,
            previousId: input.previousId,
            routineId: input.routineId,
            title: input.title,
            type: input.type,
        };
        // Create type-specific data
        switch (input.type) {
            case NodeType.Combine:
                if (Boolean(input.nodeCombineCreate) === Boolean(input.nodeCombineUpdate)) throw new CustomError(CODE.InvalidArgs, 'If type is combine, nodeCombineCreate xor nodeCombineUpdate must be provided');
                nodeData.nodeCombine = this.relationshipBuilderCombineNode(userId, input, false);
                break;
            case NodeType.Decision:
                if (Boolean(input.nodeDecisionCreate) === Boolean(input.nodeDecisionUpdate)) throw new CustomError(CODE.InvalidArgs, 'If type is decision, nodeDecisionCreate xor nodeDecisionUpdate must be provided');
                nodeData.nodeDecision = this.relationshipBuilderDecisionNode(userId, input, false);
                break;
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
            ...selectHelper<Node, node>(info, format.toDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(node)], {});
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
     * Supplemental field is role
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<Node>[],
        known: { [x: string]: any[] }, // Known values (i.e. don't need to query), in same order as objects
    ): Promise<RecursivePartial<Node>[]> {
        // If userId not provided, return the input with role set to null
        if (!userId) return objects.map(x => ({ ...x, role: null }));
        // Check is role is provided
        if (known.role) objects = objects.map((x, i) => ({ ...x, role: known.role[i] }));
        // Otherwise, query for role
        else {
            console.log('node supplemental fields', objects[0]?.routine?.owner?.__typename)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.routine?.owner?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = await OrganizationModel(prisma).getRoles(userId, organizationIds);
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.routine?.owner?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        return objects;
    },
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