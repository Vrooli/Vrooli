import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { padSelect } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkCreateInput,
    GqlUpdate: NodeLinkUpdateInput,
    GqlRelCreate: NodeLinkCreateInput,
    GqlRelUpdate: NodeLinkUpdateInput,
    GqlModel: NodeLink,
    GqlPermission: any,
    PrismaCreate: Prisma.node_linkUpsertArgs['create'],
    PrismaUpdate: Prisma.node_linkUpsertArgs['update'],
    PrismaRelCreate: Prisma.node_linkCreateWithoutRoutineVersionInput,
    PrismaRelUpdate: Prisma.node_linkUpdateWithoutRoutineVersionInput,
    PrismaModel: Prisma.node_linkGetPayload<SelectWrap<Prisma.node_linkSelect>>,
    PrismaSelect: Prisma.node_linkSelect,
    PrismaWhere: Prisma.node_linkWhereInput,
}

const __typename = 'NodeLink' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        whens: 'NodeLinkWhen',
    },
})

const mutater = (): Mutater<Model> => ({
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
    yup: { create: {} as any, update: {} as any },
})

const displayer = (): Displayer<Model> => ({
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
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})