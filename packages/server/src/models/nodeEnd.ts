import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";

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

export const NodeEndModel = ({
    delegate: (prisma: PrismaType) => prisma.node_end,
    format: formatter(),
    mutate: mutater(),
    type: 'NodeEnd' as GraphQLModelType,
})