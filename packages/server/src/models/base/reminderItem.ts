import { MaxObjects, reminderItemValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { ReminderItemFormat } from "../format/reminderItem";
import { ModelLogic } from "../types";
import { ReminderModel } from "./reminder";
import { ReminderItemModelLogic, ReminderModelLogic } from "./types";

const __typename = "ReminderItem" as const;
const suppFields = [] as const;
export const ReminderItemModel: ModelLogic<ReminderItemModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.reminder_item,
    display: {
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    },
    format: ReminderItemFormat,
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
    search: undefined,
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ReminderModel.validate.isPublic(data.reminder as ReminderModelLogic["PrismaModel"], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ReminderModel.validate.owner(data.reminder as ReminderModelLogic["PrismaModel"], userId),
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
