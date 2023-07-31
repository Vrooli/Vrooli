import { MaxObjects, reminderListValidation } from "@local/shared";
import { shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { ReminderListFormat } from "../format/reminderList";
import { ModelLogic } from "../types";
import { FocusModeModel } from "./focusMode";
import { FocusModeModelLogic, ReminderListModelLogic } from "./types";

const __typename = "ReminderList" as const;
const suppFields = [] as const;
export const ReminderListModel: ModelLogic<ReminderListModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.reminder_list,
    display: {
        label: {
            select: () => ({ id: true, focusMode: { select: FocusModeModel.display.label.select() } }),
            // Label is schedule's label
            get: (select, languages) => FocusModeModel.display.label.get(select.focusMode as FocusModeModelLogic["PrismaModel"], languages),
        },
    },
    format: ReminderListFormat,
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
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => FocusModeModel.validate.owner(data.focusMode as FocusModeModelLogic["PrismaModel"], userId),
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
