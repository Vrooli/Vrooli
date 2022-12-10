import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { bestLabel, translationRelationshipBuilder } from "../utils";
import { padSelect } from "../builders";
import { RoutineModel } from "./routine";
import { SelectWrap } from "../builders/types";

const __typename = 'NodeRoutineListItem' as const;

const suppFields = [] as const;
const formatter = (): Formatter<NodeRoutineListItem, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        routineVersion: 'RoutineVersion',
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
    Prisma.node_routine_list_itemGetPayload<SelectWrap<Prisma.node_routine_list_itemSelect>>
> => ({
    select: () => ({
        id: true,
        translations: padSelect({ id: true, name: true }),
        routineVersion: padSelect(RoutineModel.display.select),
    }),
    label: (select, languages) => {
        // Prefer item translations over routineVersion's
        const itemLabel = bestLabel(select.translations, 'name', languages);
        if (itemLabel.length > 0) return itemLabel;
        return RoutineModel.display.label(select.routineVersion as any, languages);
    }
})

export const NodeRoutineListItemModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_routine_list_item,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
})