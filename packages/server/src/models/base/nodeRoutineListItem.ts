import { MaxObjects, nodeRoutineListItemValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeRoutineListItemFormat } from "../formats";
import { NodeRoutineListItemModelInfo, NodeRoutineListItemModelLogic, NodeRoutineListModelInfo, NodeRoutineListModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic } from "./types";

const __typename = "NodeRoutineListItem" as const;
export const NodeRoutineListItemModel: NodeRoutineListItemModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list_item,
    display: {
        label: {
            select: () => ({
                id: true,
                translations: { select: { id: true, name: true } },
                routineVersion: { select: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.select() },
            }),
            get: (select, languages) => {
                // Prefer item translations over routineVersion's
                const itemLabel = bestTranslation(select.translations, languages)?.name ?? "";
                if (itemLabel.length > 0) return itemLabel;
                return ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.get(select.routineVersion as RoutineVersionModelInfo["PrismaModel"], languages);
            },
        },
    },
    format: NodeRoutineListItemFormat,
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
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, list: "NodeRoutineList" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeRoutineListModelLogic>("NodeRoutineList").validate.owner(data?.list as NodeRoutineListModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeRoutineListModelLogic>("NodeRoutineList").validate.isDeleted(data.list as NodeRoutineListModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeRoutineListItemModelInfo["PrismaSelect"]>([["list", "NodeRoutineList"]], ...rest),
        visibility: {
            private: { list: ModelMap.get<NodeRoutineListModelLogic>("NodeRoutineList").validate.visibility.private },
            public: { list: ModelMap.get<NodeRoutineListModelLogic>("NodeRoutineList").validate.visibility.public },
            owner: (userId) => ({ list: ModelMap.get<NodeRoutineListModelLogic>("NodeRoutineList").validate.visibility.owner(userId) }),
        },
    },
});
