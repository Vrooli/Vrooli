import { MaxObjects } from "@local/consts";
import { nodeRoutineListItemValidation } from "@local/validation";
import { noNull, selPad, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { RoutineModel } from "./routine";
const __typename = "NodeRoutineListItem";
const suppFields = [];
export const NodeRoutineListItemModel = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list_item,
    display: {
        select: () => ({
            id: true,
            translations: selPad({ id: true, name: true }),
            routineVersion: selPad(RoutineModel.display.select),
        }),
        label: (select, languages) => {
            const itemLabel = bestLabel(select.translations, "name", languages);
            if (itemLabel.length > 0)
                return itemLabel;
            return RoutineModel.display.label(select.routineVersion, languages);
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
        owner: (data, userId) => NodeRoutineListModel.validate.owner(data.list, userId),
        isDeleted: (data, languages) => NodeRoutineListModel.validate.isDeleted(data.list, languages),
        isPublic: (data, languages) => NodeRoutineListModel.validate.isPublic(data.list, languages),
        visibility: {
            private: { list: NodeRoutineListModel.validate.visibility.private },
            public: { list: NodeRoutineListModel.validate.visibility.public },
            owner: (userId) => ({ list: NodeRoutineListModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=nodeRoutineListItem.js.map