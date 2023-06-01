import { MaxObjects, NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput, nodeRoutineListItemValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../utils";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = "NodeRoutineListItem" as const;
const suppFields = [] as const;
export const NodeRoutineListItemModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeRoutineListItemCreateInput,
    GqlUpdate: NodeRoutineListItemUpdateInput,
    GqlModel: NodeRoutineListItem,
    GqlPermission: object,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_routine_list_itemUpsertArgs["create"],
    PrismaUpdate: Prisma.node_routine_list_itemUpsertArgs["update"],
    PrismaModel: Prisma.node_routine_list_itemGetPayload<SelectWrap<Prisma.node_routine_list_itemSelect>>,
    PrismaSelect: Prisma.node_routine_list_itemSelect,
    PrismaWhere: Prisma.node_routine_list_itemWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_routine_list_item,
    display: {
        label: {
            select: () => ({
                id: true,
                translations: selPad({ id: true, name: true }),
                routineVersion: selPad(RoutineModel.display.label.select),
            }),
            get: (select, languages) => {
                // Prefer item translations over routineVersion's
                const itemLabel = bestTranslation(select.translations, languages)?.name ?? "";
                if (itemLabel.length > 0) return itemLabel;
                return RoutineModel.display.label.get(select.routineVersion as any, languages);
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            routineVersion: "RoutineVersion",
        },
        prismaRelMap: {
            __typename,
            list: "NodeRoutineList",
            routineVersion: "RoutineVersion",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: data.index,
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "NodeRoutineList", parentRelationshipName: "list", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "nodeLists", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "RoutineVersion", parentRelationshipName: "nodeLists", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: nodeRoutineListItemValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ list: "NodeRoutineList" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeRoutineListModel.validate!.owner(data.list as any, userId),
        isDeleted: (data, languages) => NodeRoutineListModel.validate!.isDeleted(data.list as any, languages),
        isPublic: (data, languages) => NodeRoutineListModel.validate!.isPublic(data.list as any, languages),
        visibility: {
            private: { list: NodeRoutineListModel.validate!.visibility.private },
            public: { list: NodeRoutineListModel.validate!.visibility.public },
            owner: (userId) => ({ list: NodeRoutineListModel.validate!.visibility.owner(userId) }),
        },
    },
});
