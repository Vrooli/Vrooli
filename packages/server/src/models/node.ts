import { BaseModel } from "./base";
import { onlyPrimitives } from "../utils/objectTools";
import pkg from '@prisma/client';
import { PrismaSelect } from "@paljs/plugins";
import { CustomError } from "../error";
import { CODE } from "@local/shared";
const { NodeType } = pkg;

const MAX_NODES_IN_ROUTINE = 100;

export class NodeModel extends BaseModel<any, any> {
    
    constructor(prisma: any) {
        super(prisma, 'node');
    }

    /**
     * Helps create a combine node
     * @param data 
     */
    private async createCombineHelper(data: any): Promise<{ dataCombineId: string }> {
        const row = await this.prisma.nodeCombine.create({ data });
        return { dataCombineId: row.id };
    }

    /**
     * Helps create a decision node
     * @param data 
     */
    private async createDecisionHelper(data: any): Promise<{ dataDecisionId: string }> {
        const row = await this.prisma.nodeDecision.create({ data });
        return { dataDecisionId: row.id };
    }

    /**
     * Helps create a end node
     * @param data 
     */
    private async createEndHelper(data: any): Promise<{ dataEndId: string }> {
        const row = await this.prisma.nodeEnd.create({ data });
        return { dataEndId: row.id };
    }

    /**
     * Helps create a loop node
     * @param data 
     */
    private async createLoopHelper(data: any): Promise<{ dataLoopId: string }> {
        const row = await this.prisma.nodeLoop.create({ data });
        return { dataLoopId: row.id };
    }

    /**
     * Helps create a routine list node
     * @param data 
     */
    private async createRoutineListHelper(data: any): Promise<{ dataRoutineListId: string }> {
        const row = await this.prisma.nodeRoutineList.create({ data });
        return { dataRoutineListId: row.id };
    }

    /**
     * Helps create a redirect node
     * @param data 
     */
    private async createRedirectHelper(data: any): Promise<{ dataRedirectId: string }> {
        const row = await this.prisma.nodeRedirect.create({ data });
        return { dataRedirectId: row.id };
    }

    /**
     * Helps create a start node
     * @param data 
     */
    private async createStartHelper(data: any): Promise<{ dataStartId: string }> {
        const row = await this.prisma.nodeStart.create({ data });
        return { dataStartId: row.id };
    }

    async create(data: any, info: any) {
        // Check if routine ID was provided
        if (!data.routineId) throw new CustomError(CODE.InvalidArgs, 'Routine ID not specified')
        // Check if routine has reached max nodes
        const nodeCount = await this.prisma.routine.findUnique({
            where: { id: data.routineId },
            include: { _count: { select: { nodes: true } } }
        });
        if (nodeCount._count.nodes >= MAX_NODES_IN_ROUTINE) throw new CustomError(CODE.MaxNodesReached);
        // Remove relationship data, as they are handled on a case-by-case basis
        let cleanedData = onlyPrimitives(data);
        // Map node type to helper function
        const typeHelperMapper: any = {
            [NodeType.COMBINE]: this.createCombineHelper,
            [NodeType.DECISION]: this.createDecisionHelper,
            [NodeType.END]: this.createEndHelper,
            [NodeType.LOOP]: this.createLoopHelper,
            [NodeType.ROUTINE_LIST]: this.createRoutineListHelper,
            [NodeType.REDIRECT]: this.createRedirectHelper,
            [NodeType.START]: this.createStartHelper,
        }
        const typeHelper = typeHelperMapper[data.type];
        // Create type-specific data
        const typeData = await typeHelper(data.data);
        // Create base node object
        return await this.prisma.node.create({ 
            data: {
                ...cleanedData,
                ...typeData
            },
            ...(new PrismaSelect(info).value)
        });
    }
}