import { MaxObjects, nodeRoutineListItemValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { NodeRoutineListItemFormat } from "../format/nodeRoutineListItem";
import { ModelLogic } from "../types";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { RoutineModel } from "./routine";
import { NodeRoutineListItemModelLogic } from "./types";

const __typename = "NodeRoutineListItem" as const;
const suppFields = [] as const;
export const NodeRoutineListItemModel: ModelLogic<NodeRoutineListItemModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list_item,
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
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, list: "NodeRoutineList" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeRoutineListModel.validate.owner(data.list as any, userId),
        isDeleted: (data, languages) => NodeRoutineListModel.validate.isDeleted(data.list as any, languages),
        isPublic: (data, languages) => NodeRoutineListModel.validate.isPublic(data.list as any, languages),
        visibility: {
            private: { list: NodeRoutineListModel.validate.visibility.private },
            public: { list: NodeRoutineListModel.validate.visibility.public },
            owner: (userId) => ({ list: NodeRoutineListModel.validate.visibility.owner(userId) }),
        },
    },
});
