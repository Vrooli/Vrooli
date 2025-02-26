import { MaxObjects, reminderListValidation } from "@local/shared";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ReminderListFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { FocusModeModelInfo, FocusModeModelLogic, ReminderListModelLogic } from "./types.js";

const __typename = "ReminderList" as const;
export const ReminderListModel: ReminderListModelLogic = ({
    __typename,
    dbTable: "reminder_list",
    display: () => ({
        label: {
            select: () => ({ id: true, focusMode: { select: ModelMap.get<FocusModeModelLogic>("FocusMode").display().label.select() } }),
            // Label is schedule's label
            get: (select, languages) => ModelMap.get<FocusModeModelLogic>("FocusMode").display().label.get(select.focusMode as FocusModeModelInfo["DbModel"], languages),
        },
    }),
    format: ReminderListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                focusMode: await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, objectType: "FocusMode", parentRelationshipName: "reminderList", data, ...rest }),
                reminders: await shapeHelper({ relation: "reminders", relTypes: ["Create"], isOneToOne: false, objectType: "Reminder", parentRelationshipName: "reminderList", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                focusMode: await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, objectType: "FocusMode", parentRelationshipName: "reminderList", data, ...rest }),
                reminders: await shapeHelper({ relation: "reminders", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "Reminder", parentRelationshipName: "reminderList", data, ...rest }),
            }),
        },
        yup: reminderListValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<FocusModeModelLogic>("FocusMode").validate().owner(data?.focusMode as FocusModeModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            own: function getOwn(data) {
                return {
                    focusMode: useVisibility("FocusMode", "Own", data),
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("ReminderList", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("ReminderList", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
