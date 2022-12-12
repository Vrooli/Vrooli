import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../endpoints/types";
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
    GqlCreate: NodeRoutineListCreateInput,
    GqlUpdate: NodeRoutineListUpdateInput,
    GqlRelCreate: NodeRoutineListCreateInput,
    GqlRelUpdate: NodeRoutineListUpdateInput,
    GqlModel: NodeRoutineList,
    GqlPermission: any,
    PrismaCreate: Prisma.node_routine_listUpsertArgs['create'],
    PrismaUpdate: Prisma.node_routine_listUpsertArgs['update'],
    PrismaRelCreate: Prisma.node_routine_listCreateWithoutNodeInput,
    PrismaRelUpdate: Prisma.node_routine_listUpdateWithoutNodeInput,
    PrismaModel: Prisma.node_routine_listGetPayload<SelectWrap<Prisma.node_routine_listSelect>>,
    PrismaSelect: Prisma.node_routine_listSelect,
    PrismaWhere: Prisma.node_routine_listWhereInput,
}

const __typename = 'NodeRoutineList' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        items: 'NodeRoutineListItem',
    },
})

const mutater = (): Mutater<Model> => ({
    shape: {
        relCreate: async ({ data, prisma, userData }) => {
            return {
                id: data.id,
                isOrdered: data.isOrdered ?? undefined,
                isOptional: data.isOptional ?? undefined,
                routines: await relBuilderHelper({ data, isAdd: true, isRequired: false, isOneToOne: false, relationshipName: 'routines', objectType: 'NodeRoutineListItem', prisma, userData }),
            }
        },
        relUpdate: async ({ data, prisma, userData }) => {
            return {
                isOrdered: data.isOrdered ?? undefined,
                isOptional: data.isOptional ?? undefined,
                routines: await relBuilderHelper({ data, isAdd: false, isOneToOne: false, isRequired: false, relationshipName: 'routines', objectType: 'NodeRoutineListItem', prisma, userData }),
            }
        },
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