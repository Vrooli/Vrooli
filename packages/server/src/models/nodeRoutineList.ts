import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { padSelect } from "../builders";
import { NodeModel } from "./node";

const formatter = (): Formatter<NodeRoutineList, any> => ({
    relationshipMap: {
        __typename: 'NodeRoutineList',
        routines: {
            __typename: 'NodeRoutineListItem',
            routine: 'Routine',
        },
    },
})

const mutater = (): Mutater<
    NodeRoutineList,
    false,
    false,
    { graphql: NodeRoutineListCreateInput, db: Prisma.node_routine_listCreateWithoutNodeInput },
    { graphql: NodeRoutineListUpdateInput, db: Prisma.node_routine_listUpdateWithoutNodeInput }
> => ({
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
    yup: {},
})

const displayer = (): Displayer<
    Prisma.node_routine_listSelect,
    Prisma.node_routine_listGetPayload<{ select: { [K in keyof Required<Prisma.node_routine_listSelect>]: true } }>
> => ({
    select: { id: true, node: padSelect(NodeModel.display.select) },
    label: (select, languages) => NodeModel.display.label(select.node as any, languages),
})

export const NodeRoutineListModel = ({
    delegate: (prisma: PrismaType) => prisma.node_routine_list,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'NodeRoutineList' as GraphQLModelType,
})