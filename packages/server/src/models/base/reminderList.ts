import { MaxObjects, reminderListValidation } from "@local/shared";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ReminderListFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { FocusModeModelInfo, FocusModeModelLogic, ReminderListModelLogic, UserModelInfo, UserModelLogic } from "./types.js";

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
                user: { connect: { id: rest.userData.id } },
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
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<UserModelLogic>("User").validate().owner(data?.user as UserModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            own: function getOwn(data) {
                return {
                    user: useVisibility("User", "Own", data),
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
