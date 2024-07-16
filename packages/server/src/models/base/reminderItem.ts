import { MaxObjects, reminderItemValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ReminderItemFormat } from "../formats";
import { ReminderItemModelInfo, ReminderItemModelLogic, ReminderModelInfo, ReminderModelLogic } from "./types";

const __typename = "ReminderItem" as const;
export const ReminderItemModel: ReminderItemModelLogic = ({
    __typename,
    dbTable: "reminder_item",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: ReminderItemFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                isComplete: noNull(data.isComplete),
                name: data.name,
                reminder: await shapeHelper({ relation: "reminder", relTypes: ["Connect"], isOneToOne: true, objectType: "Reminder", parentRelationshipName: "reminderItems", data, ...rest }),
            }),
            update: async ({ data }) => ({
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                name: noNull(data.name),
            }),
        },
        yup: reminderItemValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ReminderItemModelInfo["PrismaSelect"]>([["reminder", "Reminder"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ReminderModelLogic>("Reminder").validate().owner(data?.reminder as ReminderModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            reminder: "Reminder",
        }),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                reminder: ModelMap.get<ReminderModelLogic>("Reminder").validate().visibility.owner(userId),
            }),
        },
    }),
});
