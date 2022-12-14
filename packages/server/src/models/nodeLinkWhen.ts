import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { translationRelationshipBuilder } from "../utils";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkWhenCreateInput,
    GqlUpdate: NodeLinkWhenUpdateInput,
    GqlModel: NodeLinkWhen,
    GqlPermission: any,
    PrismaCreate: Prisma.node_link_whenUpsertArgs['create'],
    PrismaUpdate: Prisma.node_link_whenUpsertArgs['update'],
    PrismaModel: Prisma.node_link_whenGetPayload<SelectWrap<Prisma.node_link_whenSelect>>,
    PrismaSelect: Prisma.node_link_whenSelect,
    PrismaWhere: Prisma.node_link_whenWhereInput,
}

const __typename = 'NodeLinkWhen' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

const relBase = async (prisma: PrismaType, userData: SessionUser, data: NodeLinkWhenCreateInput | NodeLinkWhenUpdateInput, isAdd: boolean) => {
    return {
        translations: await translationRelationshipBuilder(prisma, userData, data, true),
    }
}

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...await relBase(prisma, userData, data, true),
                condition: data.condition,
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                ...await relBase(prisma, userData, data, false),
                condition: data.condition ?? undefined,
            }
        },
    },
    yup: { create: {} as any, update: {} as any },
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLinkWhenModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})