import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { padSelect } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";

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

const displayer = (): Displayer<
    Prisma.node_linkSelect,
    Prisma.node_linkGetPayload<SelectWrap<Prisma.node_linkSelect>>
> => ({
    select: () => ({ 
        id: true, 
        from: padSelect(NodeModel.display.select),
        to: padSelect(NodeModel.display.select),
    }),
    // Label combines from and to labels
    label: (select, languages) => {
        return `${NodeModel.display.label(select.from as any, languages)} -> ${NodeModel.display.label(select.to as any, languages)}`
    }
})

export const NodeLinkModel = ({
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'NodeLink' as GraphQLModelType,
})