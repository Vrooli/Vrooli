import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { noNull, padSelect, shapeHelper } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeRoutineListCreateInput,
    GqlUpdate: NodeRoutineListUpdateInput,
    GqlModel: NodeRoutineList,
    GqlPermission: any,
    PrismaCreate: Prisma.node_routine_listUpsertArgs['create'],
    PrismaUpdate: Prisma.node_routine_listUpsertArgs['update'],
    PrismaModel: Prisma.node_routine_listGetPayload<SelectWrap<Prisma.node_routine_listSelect>>,
    PrismaSelect: Prisma.node_routine_listSelect,
    PrismaWhere: Prisma.node_routine_listWhereInput,
}

const __typename = 'NodeRoutineList' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        items: 'NodeRoutineListItem',
    },
    prismaRelMap: {
        __typename,
        node: 'Node',
        items: 'NodeRoutineListItem',
    }
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => ({
            id: data.id,
            isOrdered: noNull(data.isOrdered),
            isOptional: noNull(data.isOptional),
            ...(await shapeHelper({ relation: 'node', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'node', data, prisma, userData })),
            ...(await shapeHelper({ relation: 'items', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeRoutineListItem', parentRelationshipName: 'list', data, prisma, userData })),
        }),
        update: async ({ data, prisma, userData }) => ({
            isOrdered: noNull(data.isOrdered),
            isOptional: noNull(data.isOptional),
            ...(await shapeHelper({ relation: 'items', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeRoutineListItem', parentRelationshipName: 'list', data, prisma, userData })),
        }),
    },
    yup: { create: {} as any, update: {} as any },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, node: padSelect(NodeModel.display.select) }),
    label: (select, languages) => NodeModel.display.label(select.node as any, languages),
})

export const NodeRoutineListModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_routine_list,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})