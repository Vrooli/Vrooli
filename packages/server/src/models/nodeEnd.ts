import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { NodeModel } from "./node";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeEndCreateInput,
    GqlUpdate: NodeEndUpdateInput,
    GqlModel: NodeEnd,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_endUpsertArgs['create'],
    PrismaUpdate: Prisma.node_endUpsertArgs['update'],
    PrismaModel: Prisma.node_endGetPayload<SelectWrap<Prisma.node_endSelect>>,
    PrismaSelect: Prisma.node_endSelect,
    PrismaWhere: Prisma.node_endWhereInput,
}

const __typename = 'NodeEnd' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        suggestedNextRoutineVersion: 'RoutineVersion',
    },
    prismaRelMap: {
        __typename,
        suggestedNextRoutineVersion: 'RoutineVersion',
        node: 'Node',
    },
    joinMap: { suggestedNextRoutineVersion: 'toRoutineVersion' },
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data }) => {
            return {
                id: data.id,
                nodeId: data.nodeId,
                wasSuccessful: data.wasSuccessful ?? undefined,
            }
        },
        update: async ({ data }) => {
            return {
                wasSuccessful: data.wasSuccessful ?? undefined,
            }
        },
    },
    yup: { create: {} as any, update: {} as any },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, node: padSelect(NodeModel.display.select) }),
    label: (select, languages) => NodeModel.display.label(select.node as any, languages),
})

export const NodeEndModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_end,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
})