import { DeleteOneInput, Node, NodeAddInput, NodeUpdateInput, Success } from "../schema/types";
import { FormatConverter, InfoType, MODEL_TYPES, selectHelper } from "./base";
import { CustomError } from "../error";
import { CODE, nodeAdd, NodeType } from "@local/shared";
import { PrismaType, RecursivePartial } from "types";
import { node } from "@prisma/client";
import { hasProfanityRecursive } from "../utils/censor";

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
            // // Convert data field to select all data types TODO
            // if (modified.data) {
            //     modified.nodeCombine = {
            //         from: {
            //             fromId: true
            //         },
            //         to:
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
            return modified
        },
        toGraphQL: (obj: RecursivePartial<node>): RecursivePartial<Node> => {
            // Create data field from data types
            let modified: any = obj;
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

const typeMapper = {
    [NodeType.Combine]: 'nodeCombineId',
    [NodeType.Decision]: 'nodeDecisionId',
    [NodeType.End]: 'nodeEndId',
    [NodeType.Loop]: 'nodeLoopId',
    [NodeType.RoutineList]: 'nodeRoutineListId',
    [NodeType.Redirect]: 'nodeRedirectId',
    [NodeType.Start]: 'nodeStartId',
}

/**
 * Handles the authorized adding, updating, and deleting of nodes.
 */
 const noder = (format: FormatConverter<Node, node>, prisma: PrismaType) => ({
    async addNode(
        userId: string,
        input: NodeAddInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeAdd.validateSync(input, { abortEarly: false });
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
        // TODO
        // Create node
        const node = await prisma.node.create({
            data: nodeData,
            ...selectHelper<Node, node>(info, format.toDB)
        })
        // Return node
        return format.toGraphQL(node);
    },
    async updateNode(
        userId: string,
        input: NodeUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Node>> {
        // Check for valid arguments
        nodeAdd.validateSync(input, { abortEarly: false });
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
        // Update type-specific data
        // TODO
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
        return format.toGraphQL(node);
    },
    async deleteNode(userId: string, input: DeleteOneInput): Promise<Success> {
        // Check if authorized TODO
        const isAuthorized = true;
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        await prisma.node.delete({
            where: { id: input.id }
        })
        return { success: true };
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