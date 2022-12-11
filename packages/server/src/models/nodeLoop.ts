import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoop } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

const __typename = 'NodeLoop' as const;
const suppFields = [] as const;
const formatter = (): Formatter<NodeLoop, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        whiles: 'NodeLoopWhile',
    },
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<
    Prisma.node_loopSelect,
    Prisma.node_loopGetPayload<SelectWrap<Prisma.node_loopSelect>>
> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLoopModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    validate: {} as any,
})