import { MaxObjects, ReminderList, ReminderListCreateInput, ReminderListUpdateInput, reminderListValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { FocusModeModel } from "./focusMode";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "ReminderList" as const;
export const ReminderListFormat: Formatter<ModelReminderListLogic> = {
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
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                focusMode: FocusModeModel.validate!.visibility.owner(userId),
            }),
};
