import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "./actions";

const formatter = (): Formatter<NodeLink, any> => ({
    relationshipMap: {
        __typename: 'NodeLink',
    },
})

const mutater = (): Mutater<
    NodeLink,
    false,
    false,
    { graphql: NodeLinkCreateInput, db: Prisma.node_linkCreateWithoutRoutineVersionInput },
    { graphql: NodeLinkUpdateInput, db: Prisma.node_linkUpdateWithoutRoutineVersionInput }
> => ({
    shape: {
        relCreate: async ({ prisma, userData, data }) => {
            let { fromId, toId, ...rest } = data;
            return {
                ...rest,
                whens: await relBuilderHelper({ data, isAdd: true, isOneToOne: false, isRequired: false, relationshipName: 'whens', objectType: 'Routine', prisma, userData }),
                from: { connect: { id: fromId } },
                to: { connect: { id: toId } },
            }
        },
        relUpdate: async ({ prisma, userData, data }) => {
            let { fromId, toId, ...rest } = data;
            return {
                ...rest,
                whens: await relBuilderHelper({ data, isAdd: false, isOneToOne: false, isRequired: false, relationshipName: 'whens', objectType: 'Routine', prisma, userData }),
                from: fromId ? { connect: { id: fromId } } : undefined,
                to: toId ? { connect: { id: toId } } : undefined,
            }
        },
    },
    yup: {},
})

export const NodeLinkModel = ({
    delegate: (prisma: PrismaType) => prisma.node_link,
    format: formatter(),
    mutate: mutater(),
    type: 'NodeLink' as GraphQLModelType,
})