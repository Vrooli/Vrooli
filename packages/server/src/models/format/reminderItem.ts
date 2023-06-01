import { MaxObjects, ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput, reminderItemValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ReminderModel } from "./reminder";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "ReminderItem" as const;
export const ReminderItemFormat: Formatter<ModelReminderItemLogic> = {
        gqlRelMap: {
            __typename,
            reminder: "Reminder",
        },
        prismaRelMap: {
            __typename,
            reminder: "Reminder",
        },
        countFields: {},
    },
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
        permissionsSelect: () => ({
            id: true,
            reminder: "Reminder",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                reminder: ReminderModel.validate!.visibility.owner(userId),
            }),
};
