import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoopWhile } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

const __typename = 'NodeLoopWhile' as const;

const suppFields = [] as const;
const formatter = (): Formatter<NodeLoopWhile, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<
    Prisma.node_loop_whileSelect,
    Prisma.node_loop_whileGetPayload<SelectWrap<Prisma.node_loop_whileSelect>>
> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLoopWhileModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})