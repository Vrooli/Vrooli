import { MaxObjects, ReminderSortBy } from "@local/consts";
import { reminderValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { ReminderListModel } from "./reminderList";
const __typename = "Reminder";
const suppFields = [];
export const ReminderModel = ({
    __typename,
    delegate: (prisma) => prisma.reminder,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            reminderItems: "ReminderItem",
            reminderList: "ReminderList",
        },
        prismaRelMap: {
            __typename,
            reminderItems: "ReminderItem",
            reminderList: "ReminderList",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest })),
                ...(await shapeHelper({ relation: "reminderItems", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                ...(await shapeHelper({ relation: "reminderItems", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest })),
            }),
        },
        yup: reminderValidation,
    },
    search: {
        defaultSort: ReminderSortBy.DueDateAsc,
        sortBy: ReminderSortBy,
        searchFields: {
            createdTimeFrame: true,
            reminderListId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "nameWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ReminderListModel.validate.isPublic(data.reminderList, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ReminderListModel.validate.owner(data.reminderList, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            reminderList: "ReminderList",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                reminderList: ReminderListModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=reminder.js.map