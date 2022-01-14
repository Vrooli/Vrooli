import { Node, NodeDecisionItem, NodeInput } from "schema/types";
import { deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";
import { PrismaSelect } from "@paljs/plugins";
import { CustomError } from "../error";
import { CODE, NodeType } from "@local/shared";
import { onlyPrimitives } from "../utils";
import { PrismaType, RecursivePartial } from "types";

const MAX_NODES_IN_ROUTINE = 100;

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type NodeRelationshipList = 'dataCombine' | 'dataDecision' | 'dataEnd' | 'dataLoop' |
    'dataRoutineList' | 'dataRedirect' | 'dataStart' | 'previous' | 'next' | 'routine' | 'Previous' |
    'Next' | 'To' | 'From' | 'DecisionItem';
// Type 2. QueryablePrimitives
export type NodeQueryablePrimitives = Omit<Node, NodeRelationshipList>;
// Type 3. AllPrimitives
export type NodeAllPrimitives = NodeQueryablePrimitives;
// type 4. Database shape
export type NodeDB = NodeAllPrimitives &
    Pick<Node, 'previous' | 'next' | 'routine' | 'Previous' | 'Next'> &
{
    dataCombine?: { from: Node[], to: Node },
    dataDecision?: NodeDecisionItem[],
    dataEnd?: {}, //TODO
    dataLoop?: {}, //TODO
    dataRoutineList?: {}, //TODO
    dataRedirect?: {}, //TODO
    dataStart?: {}, //TODO
    To: {}[] //TODO
    From: {}[] //TODO
    DecisionItem?: NodeDecisionItem[]
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Node, NodeDB> => ({
    toDB: (obj: RecursivePartial<Node>): RecursivePartial<NodeDB> => ({ ...obj }), //TODO
    toGraphQL: (obj: RecursivePartial<NodeDB>): RecursivePartial<Node> => ({ ...obj }) //TODO must at least convert previous and next to ids
})

/**
 * Custom compositional component for creating nodes
 * @param state 
 * @returns 
 */
const creater = (prisma?: PrismaType) => ({
    async createCombineHelper(data: any): Promise<{ dataCombineId: string }> {
        const row = await prisma?.node_combine.create({ data });
        return { dataCombineId: row?.id ?? '' };
    },
    async createDecisionHelper(data: any): Promise<{ dataDecisionId: string }> {
        const row = await prisma?.node_decision.create({ data });
        return { dataDecisionId: row?.id ?? '' };
    },
    async createEndHelper(data: any): Promise<{ dataEndId: string }> {
        const row = await prisma?.node_end.create({ data });
        return { dataEndId: row?.id ?? '' };
    },
    async createLoopHelper(data: any): Promise<{ dataLoopId: string }> {
        const row = await prisma?.node_loop.create({ data });
        return { dataLoopId: row?.id ?? '' };
    },
    async createRoutineListHelper(data: any): Promise<{ dataRoutineListId: string }> {
        const row = await prisma?.node_routine_list.create({ data });
        return { dataRoutineListId: row?.id ?? '' };
    },
    async createRedirectHelper(data: any): Promise<{ dataRedirectId: string }> {
        const row = await prisma?.node_redirect.create({ data });
        return { dataRedirectId: row?.id ?? '' };
    },
    async createStartHelper(data: any): Promise<{ dataStartId: string }> {
        const row = await prisma?.node_start.create({ data });
        return { dataStartId: row?.id ?? '' };
    },
    async create(data: NodeInput, info: any): Promise<NodeDB> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!data.routineId) throw new CustomError(CODE.InvalidArgs, 'Routine ID not specified')
        if (!data.type) throw new CustomError(CODE.InvalidArgs, 'Node type not specified')
        // Check if routine has reached max nodes
        const nodeCount = await prisma.routine.findUnique({
            where: { id: data.routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if (!nodeCount) throw new CustomError(CODE.ErrorUnknown);
        if (nodeCount._count.nodes >= MAX_NODES_IN_ROUTINE) throw new CustomError(CODE.MaxNodesReached);
        // Remove relationship data, as they are handled on a case-by-case basis
        let cleanedData = onlyPrimitives(data);
        // Map node type to helper function and correct data field
        const typeMapper: any = {
            [NodeType.Combine]: [this.createCombineHelper, 'combineData'],
            [NodeType.Decision]: [this.createDecisionHelper, 'decisionData'],
            [NodeType.End]: [this.createEndHelper, 'endData'],
            [NodeType.Loop]: [this.createLoopHelper, 'loopData'],
            [NodeType.RoutineList]: [this.createRoutineListHelper, 'routineListData'],
            [NodeType.Redirect]: [this.createRedirectHelper, 'redirectData'],
            [NodeType.Start]: [this.createStartHelper, 'startData'],
        }
        const mapResult: [any, keyof NodeInput] = typeMapper[data.type];
        // Create type-specific data
        const typeData = await mapResult[0](data[mapResult[1]]);
        // Create base node object
        return await prisma.node.create({
            data: {
                ...cleanedData,
                ...typeData
            },
            ...(new PrismaSelect(info).value)
        }) as any;
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function NodeModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Node;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...findByIder<Node, NodeDB>(model, format.toDB, prisma),
        ...creater(prisma),
        ...updater<NodeInput, Node, NodeDB>(model, format.toDB, prisma),
        ...deleter(model, prisma)
    }
}

//==============================================================
/* #endregion Model */
//==============================================================