import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { noNull, shapeHelper } from "../builders";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkWhenCreateInput,
    GqlUpdate: NodeLinkWhenUpdateInput,
    GqlModel: NodeLinkWhen,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_link_whenUpsertArgs['create'],
    PrismaUpdate: Prisma.node_link_whenUpsertArgs['update'],
    PrismaModel: Prisma.node_link_whenGetPayload<SelectWrap<Prisma.node_link_whenSelect>>,
    PrismaSelect: Prisma.node_link_whenSelect,
    PrismaWhere: Prisma.node_link_whenWhereInput,
}

const __typename = 'NodeLinkWhen' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        link: 'NodeLink',
    }
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => ({
            id: data.id,
            condition: data.condition,
            ...(await shapeHelper({ relation: 'link', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'NodeLink', parentRelationshipName: 'link', data, prisma, userData })),
            ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
        }),
        update: async ({ data, prisma, userData }) => ({
            condition: noNull(data.condition),
            ...(await shapeHelper({ relation: 'link', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'NodeLink', parentRelationshipName: 'link', data, prisma, userData })),
            ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
        }),
    },
    yup: { create: {} as any, update: {} as any },
})

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true }),
    label: () => ''
})

export const NodeLinkWhenModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
})