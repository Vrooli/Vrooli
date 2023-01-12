import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoopWhile, NodeLoopWhileCreateInput, NodeLoopWhileUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'NodeLoopWhile' as const;

const suppFields = [] as const;
export const NodeLoopWhileModel: ModelLogic<{
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
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    // Doesn't make sense to have a displayer for this model
    display: {
        select: () => ({ id: true }),
        label: () => ''
    },
    format: {
        gqlRelMap: {
            type,
        },
        prismaRelMap: {
            type,
            loop: 'NodeLoop',
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {} as any,
})