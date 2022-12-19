import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoop, NodeLoopCreateInput, NodeLoopUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLoopCreateInput,
    GqlUpdate: NodeLoopUpdateInput,
    GqlModel: NodeLoop,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_loopUpsertArgs['create'],
    PrismaUpdate: Prisma.node_loopUpsertArgs['update'],
    PrismaModel: Prisma.node_loopGetPayload<SelectWrap<Prisma.node_loopSelect>>,
    PrismaSelect: Prisma.node_loopSelect,
    PrismaWhere: Prisma.node_loopWhereInput,
}

const __typename = 'NodeLoop' as const;
const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        whiles: 'NodeLoopWhile',
    },
    prismaRelMap: {
        __typename,
        node: 'Node',
        whiles: 'NodeLoopWhile',
    }
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLoopModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    validate: {} as any,
})