import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { translationRelationshipBuilder } from "../utils";

const formatter = (): Formatter<NodeLinkWhen, any> => ({
    relationshipMap: {
        __typename: 'NodeLinkWhen',
    },
})

const relBase = async (prisma: PrismaType, userData: SessionUser, data: NodeLinkWhenCreateInput | NodeLinkWhenUpdateInput, isAdd: boolean) => {
    return {
        translations: await translationRelationshipBuilder(prisma, userData, data, true),
    }
}

const mutater = (): Mutater<
    NodeLinkWhen,
    false,
    false,
    { graphql: NodeLinkWhenCreateInput, db: Prisma.node_link_whenCreateWithoutLinkInput },
    { graphql: NodeLinkWhenUpdateInput, db: Prisma.node_link_whenUpdateWithoutLinkInput }
> => ({
    shape: {
        relCreate: async ({ data, prisma, userData }) => {
            return {
                ...await relBase(prisma, userData, data, true),
                condition: data.condition,
            }
        },
        relUpdate: async ({ data, prisma, userData }) => {
            return {
                ...await relBase(prisma, userData, data, false),
                condition: data.condition ?? undefined,
            }
        },
    },
    yup: {},
})

export const NodeLinkWhenModel = ({
    delegate: (prisma: PrismaType) => prisma.node_link,
    format: formatter(),
    mutate: mutater(),
    type: 'NodeLinkWhen' as GraphQLModelType,
})