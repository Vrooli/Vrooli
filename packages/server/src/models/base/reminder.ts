import { MaxObjects, ReminderSortBy, reminderValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { ReminderFormat } from "../formats";
import { ReminderListModelInfo, ReminderListModelLogic, ReminderModelInfo, ReminderModelLogic } from "./types";

const __typename = "Reminder" as const;
export const ReminderModel: ReminderModelLogic = ({
    __typename,
    dbTable: "reminder",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
        embed: {
            select: () => ({ id: true, embeddingNeedsUpdate: true, name: true, description: true }),
            get: ({ description, name }, languages) => {
                return getEmbeddableString({ description, name }, languages[0]);
            },
        },
    }),
    format: ReminderFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                embeddingNeedsUpdate: true,
                name: data.name,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                reminderList: await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest }),
                reminderItems: await shapeHelper({ relation: "reminderItems", relTypes: ["Create"], isOneToOne: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                embeddingNeedsUpdate: (typeof data.name === "string" || typeof data.description === "string") ? true : undefined,
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                reminderList: await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest }),
                reminderItems: await shapeHelper({ relation: "reminderItems", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest }),
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
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ReminderModelInfo["PrismaSelect"]>([["reminderList", "ReminderList"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ReminderListModelLogic>("ReminderList").validate().owner(data?.reminderList as ReminderListModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            reminderList: "ReminderList",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    reminderList: useVisibility("ReminderList", "Own", data),
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Reminder", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Reminder", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
