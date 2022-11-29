import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { bestLabel, translationRelationshipBuilder } from "../utils";
import { padSelect } from "../builders";
import { RoutineModel } from "./routine";

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
                routineVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
            }
        },
        relUpdate: async ({ data, prisma, userData }) => {
            return {
                index: data.index ?? undefined,
                isOptional: data.isOptional ?? undefined,
                routineVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: false, linkVersion: true, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: {},
})

const displayer = (): Displayer<
    Prisma.node_routine_list_itemSelect,
    Prisma.node_routine_list_itemGetPayload<{ select: { [K in keyof Required<Prisma.node_routine_list_itemSelect>]: true } }>
> => ({
    select: {
        id: true,
        translations: padSelect({ id: true, title: true }),
        routineVersion: padSelect(RoutineModel.display.select),
    },
    label: (select, languages) => {
        // Prefer item translations over routineVersion's
        const itemLabel = bestLabel(select.translations, 'title', languages);
        if (itemLabel.length > 0) return itemLabel;
        return RoutineModel.display.label(select.routineVersion as any, languages);
    }
})

export const NodeRoutineListItemModel = ({
    delegate: (prisma: PrismaType) => prisma.node_routine_list_item,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    type: 'NodeRoutineListItem' as GraphQLModelType,
})