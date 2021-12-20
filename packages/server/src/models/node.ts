import { Node, NodeInput } from "schema/types";
import { BaseState, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";
import pkg from '@prisma/client';
import { PrismaSelect } from "@paljs/plugins";
import { CustomError } from "../error";
import { CODE } from "@local/shared";
import { onlyPrimitives } from "../utils";
const { NodeType } = pkg;

const MAX_NODES_IN_ROUTINE = 100;

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})


/**
 * Custom compositional component for creating nodes
 * @param state 
 * @returns 
 */
 const creater = (state: any) => ({
    async createCombineHelper(data: any): Promise<{ dataCombineId: string }> {
        const row = await state.prisma.nodeCombine.create({ data });
        return { dataCombineId: row.id };
    },
    async createDecisionHelper(data: any): Promise<{ dataDecisionId: string }> {
        const row = await state.prisma.nodeDecision.create({ data });
        return { dataDecisionId: row.id };
    },
    async createEndHelper(data: any): Promise<{ dataEndId: string }> {
        const row = await state.prisma.nodeEnd.create({ data });
        return { dataEndId: row.id };
    },
    async createLoopHelper(data: any): Promise<{ dataLoopId: string }> {
        const row = await state.prisma.nodeLoop.create({ data });
        return { dataLoopId: row.id };
    },
    async createRoutineListHelper(data: any): Promise<{ dataRoutineListId: string }> {
        const row = await state.prisma.nodeRoutineList.create({ data });
        return { dataRoutineListId: row.id };
    },
    async createRedirectHelper(data: any): Promise<{ dataRedirectId: string }> {
        const row = await state.prisma.nodeRedirect.create({ data });
        return { dataRedirectId: row.id };
    },
    async createStartHelper(data: any): Promise<{ dataStartId: string }> {
        const row = await state.prisma.nodeStart.create({ data });
        return { dataStartId: row.id };
    },
    async create(data: NodeInput, info: any): Promise<Node> {
        // Check if routine ID was provided
        if (!data.routineId) throw new CustomError(CODE.InvalidArgs, 'Routine ID not specified')
        if (!data.type) throw new CustomError(CODE.InvalidArgs, 'Node type not specified')
        // Check if routine has reached max nodes
        const nodeCount = await state.prisma.routine.findUnique({
            where: { id: data.routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if (nodeCount._count.nodes >= MAX_NODES_IN_ROUTINE) throw new CustomError(CODE.MaxNodesReached);
        // Remove relationship data, as they are handled on a case-by-case basis
        let cleanedData = onlyPrimitives(data);
        // Map node type to helper function and correct data field
        const typeMapper: any = {
            [NodeType.COMBINE]: [this.createCombineHelper, 'combineData'],
            [NodeType.DECISION]: [this.createDecisionHelper, 'decisionData'],
            [NodeType.END]: [this.createEndHelper, 'endData'],
            [NodeType.LOOP]: [this.createLoopHelper, 'loopData'],
            [NodeType.ROUTINE_LIST]: [this.createRoutineListHelper, 'routineListData'],
            [NodeType.REDIRECT]: [this.createRedirectHelper, 'redirectData'],
            [NodeType.START]: [this.createStartHelper, 'startData'],
        }
        const mapResult: [any, keyof NodeInput] = typeMapper[data.type];
        // Create type-specific data
        const typeData = await mapResult[0](data[mapResult[1]]);
        // Create base node object
        return await state.prisma.node.create({ 
            data: {
                ...cleanedData,
                ...typeData
            },
            ...(new PrismaSelect(info).value)
        });
    }
})

export function NodeModel(prisma: any) {
    let obj: BaseState<Node> = {
        prisma,
        model: MODEL_TYPES.Node,
        format: formatter(),
    }
    
    return {
        ...obj,
        ...findByIder<Node>(obj),
        ...formatter(),
        ...creater(obj),
        ...updater<NodeInput, Node>(obj),
        ...deleter(obj)
    }
}