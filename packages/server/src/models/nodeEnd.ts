import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { NodeModel } from "./node";
import { padSelect } from "../builders";

const formatter = (): Formatter<NodeEnd, any> => ({
    relationshipMap: {
        __typename: 'NodeEnd',
    },
})

const mutater = (): Mutater<
    NodeEnd,
    false,
    false,
    { graphql: NodeEndCreateInput, db: Prisma.node_endCreateWithoutNodeInput },
    { graphql: NodeEndUpdateInput, db: Prisma.node_endUpdateWithoutNodeInput }
> => ({
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
    yup: {},
})

const displayer = (): Displayer<
    Prisma.node_endSelect,
    Prisma.node_endGetPayload<{ select: { [K in keyof Required<Prisma.node_endSelect>]: true } }>
> => ({
    select: { id: true, node: padSelect(NodeModel.display.select) },
    label: (select, languages) => NodeModel.display.label(select.node as any, languages),
})

export const NodeEndModel = ({
    delegate: (prisma: PrismaType) => prisma.node_end,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'NodeEnd' as GraphQLModelType,
})