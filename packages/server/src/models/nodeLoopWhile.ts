import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoopWhile, NodeLoopWhileCreateInput, NodeLoopWhileUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLoopWhileCreateInput,
    GqlUpdate: NodeLoopWhileUpdateInput,
    GqlModel: NodeLoopWhile,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_loop_whileUpsertArgs['create'],
    PrismaUpdate: Prisma.node_loop_whileUpsertArgs['update'],
    PrismaModel: Prisma.node_loop_whileGetPayload<SelectWrap<Prisma.node_loop_whileSelect>>,
    PrismaSelect: Prisma.node_loop_whileSelect,
    PrismaWhere: Prisma.node_loop_whileWhereInput,
}

const __typename = 'NodeLoopWhile' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        loop: 'NodeLoop',
    }
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLoopWhileModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    validate: {} as any,
})