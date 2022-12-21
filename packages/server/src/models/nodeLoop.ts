import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoop, NodeLoopCreateInput, NodeLoopUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'NodeLoop' as const;
const suppFields = [] as const;
export const NodeLoopModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop,
    // Doesn't make sense to have a displayer for this model
    display: {
        select: () => ({ id: true }),
        label: () => ''
    },
    format: {
        gqlRelMap: {
            __typename,
            whiles: 'NodeLoopWhile',
        },
        prismaRelMap: {
            __typename,
            node: 'Node',
            whiles: 'NodeLoopWhile',
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {} as any,
})