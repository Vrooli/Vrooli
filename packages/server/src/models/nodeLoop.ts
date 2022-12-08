import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const NodeLoopModel = ({
    delegate: (prisma: PrismaType) => prisma.node_loop,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'NodeLoop' as GraphQLModelType,
    validate: {} as any,
})