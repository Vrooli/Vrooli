import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { relBuilderHelper } from "../actions";
import { bestLabel, translationRelationshipBuilder } from "../utils";
import { padSelect } from "../builders";
import { RoutineModel } from "./routine";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeRoutineListItemCreateInput,
    GqlUpdate: NodeRoutineListItemUpdateInput,
    GqlModel: NodeRoutineListItem,
    GqlPermission: any,
    PrismaCreate: Prisma.node_routine_list_itemUpsertArgs['create'],
    PrismaUpdate: Prisma.node_routine_list_itemUpsertArgs['update'],
    PrismaModel: Prisma.node_routine_list_itemGetPayload<SelectWrap<Prisma.node_routine_list_itemSelect>>,
    PrismaSelect: Prisma.node_routine_list_itemSelect,
    PrismaWhere: Prisma.node_routine_list_itemWhereInput,
}

const __typename = 'NodeRoutineListItem' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        routineVersion: 'RoutineVersion',
    },
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                id: data.id,
                index: data.index,
                isOptional: data.isOptional ?? false,
                routineVersion: await relBuilderHelper({ data, isAdd: true, isOneToOne: true, isRequired: true, linkVersion: true, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                index: data.index ?? undefined,
                isOptional: data.isOptional ?? undefined,
                routineVersion: await relBuilderHelper({ data, isAdd: false, isOneToOne: true, isRequired: false, linkVersion: true, relationshipName: 'routineVersion', objectType: 'Routine', prisma, userData }),
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        },
    },
    yup: { create: {} as any, update: {} as any },
})

const displayer = (): Displayer<Model> => ({
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