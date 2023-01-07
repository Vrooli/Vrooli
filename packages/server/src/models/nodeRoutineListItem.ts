import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { bestLabel, translationShapeHelper } from "../utils";
import { noNull, padSelect, shapeHelper } from "../builders";
import { RoutineModel } from "./routine";
import { SelectWrap } from "../builders/types";

const type = 'NodeRoutineListItem' as const;

const suppFields = [] as const;
export const NodeRoutineListItemModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeRoutineListItemCreateInput,
    GqlUpdate: NodeRoutineListItemUpdateInput,
    GqlModel: NodeRoutineListItem,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_routine_list_itemUpsertArgs['create'],
    PrismaUpdate: Prisma.node_routine_list_itemUpsertArgs['update'],
    PrismaModel: Prisma.node_routine_list_itemGetPayload<SelectWrap<Prisma.node_routine_list_itemSelect>>,
    PrismaSelect: Prisma.node_routine_list_itemSelect,
    PrismaWhere: Prisma.node_routine_list_itemWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.node_routine_list_item,
    display: {
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
    },
    format: {
        gqlRelMap: {
            type,
            routineVersion: 'RoutineVersion',
        },
        prismaRelMap: {
            type,
            list: 'NodeRoutineList',
            routineVersion: 'RoutineVersion',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                index: data.index,
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: 'list', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'NodeRoutineList', parentRelationshipName: 'list', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'RoutineVersion', parentRelationshipName: 'nodeLists', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                index: noNull(data.index),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Update'], isOneToOne: true, isRequired: false, objectType: 'RoutineVersion', parentRelationshipName: 'nodeLists', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
            }),
        },
        yup: { create: {} as any, update: {} as any },
    },
})