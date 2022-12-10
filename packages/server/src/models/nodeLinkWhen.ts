import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { translationRelationshipBuilder } from "../utils";
import { SelectWrap } from "../builders/types";

const __typename = 'NodeLinkWhen' as const;

const suppFields = [] as const;
const formatter = (): Formatter<NodeLinkWhen, typeof suppFields> => ({
    relationshipMap: {
        __typename,
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

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<
    Prisma.node_link_whenSelect,
    Prisma.node_link_whenGetPayload<SelectWrap<Prisma.node_link_whenSelect>>
> => ({
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