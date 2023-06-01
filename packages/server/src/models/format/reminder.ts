import { MaxObjects, Reminder, ReminderCreateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput, reminderValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, getEmbeddableString } from "../../utils";
import { ReminderListModel } from "./reminderList";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Reminder" as const;
export const ReminderFormat: Formatter<ModelReminderLogic> = {
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
                embeddingNeedsUpdate: true,
                name: data.name,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest })),
                ...(await shapeHelper({ relation: "reminderItems", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                embeddingNeedsUpdate: typeof data.name === "string" || typeof data.description === "string",
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                ...(await shapeHelper({ relation: "reminderItems", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest })),
            }),
        searchFields: {
            createdTimeFrame: true,
            reminderListId: true,
            updatedTimeFrame: true,
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "nameWrapped",
            ],
        permissionsSelect: () => ({
            id: true,
            reminderList: "ReminderList",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                reminderList: ReminderListModel.validate!.visibility.owner(userId),
            }),
};
