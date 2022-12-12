import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { NodeModel } from "./node";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeEndCreateInput,
    GqlUpdate: NodeEndUpdateInput,
    GqlRelCreate: NodeEndCreateInput,
    GqlRelUpdate: NodeEndUpdateInput,
    GqlModel: NodeEnd,
    GqlPermission: any,
    PrismaCreate: Prisma.node_endUpsertArgs['create'],
    PrismaUpdate: Prisma.node_endUpsertArgs['update'],
    PrismaRelCreate: Prisma.node_endCreateWithoutNodeInput,
    PrismaRelUpdate: Prisma.node_endUpdateWithoutNodeInput,
    PrismaModel: Prisma.node_endGetPayload<SelectWrap<Prisma.node_endSelect>>,
    PrismaSelect: Prisma.node_endSelect,
    PrismaWhere: Prisma.node_endWhereInput,
}

const __typename = 'NodeEnd' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

const mutater = (): Mutater<Model> => ({
    shape: {
        relCreate: async ({ data }) => {
            return {
                id: data.id,
                wasSuccessful: data.wasSuccessful ?? undefined,
            }
        },
        relUpdate: async ({ data }) => {
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

export const NodeEndModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_end,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})