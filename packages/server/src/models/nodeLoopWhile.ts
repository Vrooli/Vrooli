import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<
    Prisma.node_loop_whileSelect,
    Prisma.node_loop_whileGetPayload<SelectWrap<Prisma.node_loop_whileSelect>>
> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLoopWhileModel = ({
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'NodeLoopWhile' as GraphQLModelType,
    validate: {} as any,
})