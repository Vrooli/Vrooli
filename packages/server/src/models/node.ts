import { DeleteOneInput, Node, NodeCreateInput, NodeUpdateInput, Success } from "../schema/types";
import { addOwnerField, FormatConverter, InfoType, MODEL_TYPES, removeOwnerField, selectHelper } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, nodeCreate, NodeType, nodeUpdate } from "@local/shared";
import { PrismaType, RecursivePartial } from "types";
import { node } from "@prisma/client";
import { hasProfanityRecursive } from "../utils/censor";
import { OrganizationModel } from "./organization";

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
        // TODO
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