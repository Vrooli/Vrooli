import { MaxObjects } from "@local/consts";
import { reminderItemValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { ReminderModel } from "./reminder";
const __typename = "ReminderItem";
const suppFields = [];
export const ReminderItemModel = ({
    __typename,
    delegate: (prisma) => prisma.reminder_item,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            reminder: "Reminder",
        },
        prismaRelMap: {
            __typename,
            reminder: "Reminder",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                name: data.name,
                ...(await shapeHelper({ relation: "reminder", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Reminder", parentRelationshipName: "reminderItems", data, ...rest })),
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ReminderModel.validate.isPublic(data.reminder, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ReminderModel.validate.owner(data.reminder, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            reminder: "Reminder",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                reminder: ReminderModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=reminderItem.js.map