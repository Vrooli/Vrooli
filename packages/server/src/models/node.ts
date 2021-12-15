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

    async createCombineNode(data: any) {
    }

    async createDecisionNode(data: any) {
    }

    async createEndNode(data: any) {
    }

    async createLoopNode(data: any) {
    }

    async createRoutineListNode(data: any) {
    }

    async createRedirectNode(data: any) {
    }

    async createStartNode(data: any) {
    }

    async createNode(data: any, info: any) {
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
            [NodeType.COMBINE]: this.createCombineNode,
            [NodeType.DECISION]: this.createDecisionNode,
            [NodeType.END]: this.createEndNode,
            [NodeType.LOOP]: this.createLoopNode,
            [NodeType.ROUTINE_LIST]: this.createRoutineListNode,
            [NodeType.REDIRECT]: this.createRedirectNode,
            [NodeType.START]: this.createStartNode,
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