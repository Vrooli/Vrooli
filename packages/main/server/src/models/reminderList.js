import { MaxObjects } from "@local/consts";
import { reminderListValidation } from "@local/validation";
import { shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { FocusModeModel } from "./focusMode";
const __typename = "ReminderList";
const suppFields = [];
export const ReminderListModel = ({
    __typename,
    delegate: (prisma) => prisma.reminder_list,
    display: {
        select: () => ({ id: true, focusMode: { select: FocusModeModel.display.select() } }),
        label: (select, languages) => FocusModeModel.display.label(select.focusMode, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            focusMode: "FocusMode",
            reminders: "Reminder",
        },
        prismaRelMap: {
            __typename,
            focusMode: "FocusMode",
            reminders: "Reminder",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ...(await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "FocusMode", parentRelationshipName: "reminderList", data, ...rest })),
                ...(await shapeHelper({ relation: "reminders", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "Reminder", parentRelationshipName: "reminderList", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "FocusMode", parentRelationshipName: "reminderList", data, ...rest })),
                ...(await shapeHelper({ relation: "reminders", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Reminder", parentRelationshipName: "reminderList", data, ...rest })),
            }),
        },
        yup: reminderListValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => FocusModeModel.validate.owner(data.focusMode, userId),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                focusMode: FocusModeModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=reminderList.js.map