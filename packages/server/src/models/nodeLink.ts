import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, ModelLogic, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { noNull, padSelect, shapeHelper } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkCreateInput,
    GqlUpdate: NodeLinkUpdateInput,
    GqlModel: NodeLink,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_linkUpsertArgs['create'],
    PrismaUpdate: Prisma.node_linkUpsertArgs['update'],
    PrismaModel: Prisma.node_linkGetPayload<SelectWrap<Prisma.node_linkSelect>>,
    PrismaSelect: Prisma.node_linkSelect,
    PrismaWhere: Prisma.node_linkWhereInput,
}

const __typename = 'NodeLink' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        whens: 'NodeLinkWhen',
    },
    prismaRelMap: {
        __typename,
        from: 'Node',
        to: 'Node',
        routineVersion: 'RoutineVersion',
        whens: 'NodeLinkWhen',
    }
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ prisma, userData, data }) => ({
            id: data.id,
            operation: noNull(data.operation),
            ...(await shapeHelper({ relation: 'from', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'next', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'to', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'previous', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'RoutineVersion', parentRelationshipName: 'nodeLinks', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'whens', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeLinkWhen', parentRelationshipName: 'link', data, prisma, userData })),

        }),
        update: async ({ prisma, userData, data }) => ({
            operation: noNull(data.operation),
            ...(await shapeHelper({ relation: 'from', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Node', parentRelationshipName: 'next', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'to', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Node', parentRelationshipName: 'previous', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'whens', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeLinkWhen', parentRelationshipName: 'link', data, prisma, userData })),
        }),
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

export const NodeLinkModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
})