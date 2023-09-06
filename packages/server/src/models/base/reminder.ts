import { MaxObjects, ReminderSortBy, reminderValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, getEmbeddableString } from "../../utils";
import { ReminderFormat } from "../format/reminder";
import { ModelLogic } from "../types";
import { ReminderListModel } from "./reminderList";
import { ReminderListModelLogic, ReminderModelLogic } from "./types";

const __typename = "Reminder" as const;
const suppFields = [] as const;
export const ReminderModel: ModelLogic<ReminderModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.reminder,
    display: {
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
    },
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
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest })),
                ...(await shapeHelper({ relation: "reminderItems", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ReminderItem", parentRelationshipName: "reminder", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                embeddingNeedsUpdate: (typeof data.name === "string" || typeof data.description === "string") ? true : undefined,
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                ...(await shapeHelper({ relation: "reminderList", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: false, objectType: "ReminderList", parentRelationshipName: "reminders", data, ...rest })),
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
        isPublic: (data, languages) => ReminderListModel.validate.isPublic(data.reminderList as ReminderListModelLogic["PrismaModel"], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ReminderListModel.validate.owner(data.reminderList as ReminderListModelLogic["PrismaModel"], userId),
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
