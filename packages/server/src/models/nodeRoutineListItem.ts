import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "./actions";
import { translationRelationshipBuilder } from "./utils";

const formatter = (): Formatter<NodeRoutineListItem, any> => ({
    relationshipMap: {
        __typename: 'NodeRoutineListItem',
        routineVersion: 'Routine',
    },
})

const mutater = (): Mutater<
    NodeRoutineListItem,
    false,
    false,
    { graphql: NodeRoutineListItemCreateInput, db: Prisma.node_routine_list_itemCreateWithoutListInput },
    { graphql: NodeRoutineListItemUpdateInput, db: Prisma.node_routine_list_itemUpdateWithoutListInput }
> => ({
    shape: {
        relCreate: async ({ data, prisma, userData }) => {
            return {
                id: data.id,
                index: data.index,
                isOptional: data.isOptional ?? false,
                routineVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
            }
        },
        relUpdate: async ({ data, prisma, userData }) => {
            return {
                index: data.index ?? undefined,
                isOptional: data.isOptional ?? undefined,
                routineVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: false, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: {},
})

export const NodeRoutineListItemModel = ({
    delegate: (prisma: PrismaType) => prisma.node_routine_list_item,
    format: formatter(),
    mutate: mutater(),
    type: 'NodeRoutineListItem' as GraphQLModelType,
})