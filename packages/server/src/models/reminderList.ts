import { MaxObjects, ReminderList, ReminderListCreateInput, ReminderListUpdateInput, reminderListValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { FocusModeModel } from "./focusMode";
import { ModelLogic } from "./types";

const __typename = "ReminderList" as const;
const suppFields = [] as const;
export const ReminderListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderListCreateInput,
    GqlUpdate: ReminderListUpdateInput,
    GqlModel: ReminderList,
    GqlSearch: undefined,
    GqlSort: undefined,
    GqlPermission: object,
    PrismaCreate: Prisma.reminder_listUpsertArgs["create"],
    PrismaUpdate: Prisma.reminder_listUpsertArgs["update"],
    PrismaModel: Prisma.reminder_listGetPayload<SelectWrap<Prisma.reminder_listSelect>>,
    PrismaSelect: Prisma.reminder_listSelect,
    PrismaWhere: Prisma.reminder_listWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_list,
    display: {
        label: {
            select: () => ({ id: true, focusMode: { select: FocusModeModel.display.label.select() } }),
            // Label is schedule's label
            get: (select, languages) => FocusModeModel.display.label.get(select.focusMode as any, languages),
        },
    },
    format: {
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
        },
        yup: reminderListValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => FocusModeModel.validate!.owner(data.focusMode as any, userId),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                focusMode: FocusModeModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
