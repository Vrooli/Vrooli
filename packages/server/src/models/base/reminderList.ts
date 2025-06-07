import { DEFAULT_LANGUAGE, MaxObjects, reminderListValidation } from "@vrooli/shared";
import i18next from "i18next";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ReminderListFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type ReminderListModelLogic, type UserModelInfo, type UserModelLogic } from "./types.js";

const __typename = "ReminderList" as const;
export const ReminderListModel: ReminderListModelLogic = ({
    __typename,
    dbTable: "reminder_list",
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: (_select, languages) => i18next.t("common:Reminder", { lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE, count: 2 }),
        },
    }),
    format: ReminderListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: BigInt(data.id),
                reminders: await shapeHelper({ relation: "reminders", relTypes: ["Create"], isOneToOne: false, objectType: "Reminder", parentRelationshipName: "reminderList", data, ...rest }),
                user: { connect: { id: rest.userData.id } },
            }),
            update: async ({ data, ...rest }) => ({
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
